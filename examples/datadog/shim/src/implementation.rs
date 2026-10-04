use cocoon_guest::{Error, Result};
use libdd_ddsketch::DDSketch;
use libdd_trace_obfuscation::{
    obfuscate::obfuscate_v04_span, obfuscation_config::ObfuscationConfig, sql::obfuscate_sql_string,
};
use libdd_trace_utils::{msgpack_decoder, msgpack_encoder, span::SliceData};

#[derive(Default)]
pub struct Shim {
    configuration: ObfuscationConfig,
}

impl crate::cocoon_gen::API for Shim {
    type Sketch = DDSketch;

    fn obfuscate_sql(&mut self, query: String) -> Result<String> {
        obfuscate_sql_string(&query).map_err(|error| Error::application(error.to_string()))
    }

    fn obfuscate_traces(&mut self, payload: Vec<u8>) -> Result<Vec<u8>> {
        let mut buffer = msgpack_decoder::decode::buffer::Buffer::<SliceData<'_>>::new(&payload);
        let (mut traces, _) = msgpack_decoder::v04::from_buffer(&mut buffer)
            .map_err(|error| Error::argument(error.to_string()))?;
        if !buffer.is_empty() {
            return Err(Error::argument("trailing trace payload bytes"));
        }
        for trace in &mut traces {
            for span in trace {
                obfuscate_v04_span(span, &self.configuration);
            }
        }
        Ok(msgpack_encoder::v04::to_vec_from_v04(&traces))
    }

    fn sketch_new(&mut self) -> DDSketch {
        DDSketch::default()
    }

    fn sketch_add(&mut self, sketch: &mut DDSketch, value: f64) -> Result<()> {
        validate_point(value)?;
        sketch
            .add(value)
            .map_err(|error| Error::application(error.to_string()))
    }

    fn sketch_add_many(&mut self, sketch: &mut DDSketch, values: Vec<f64>) -> Result<()> {
        // Validate the entire batch before changing the sketch.
        for value in &values {
            validate_point(*value)?;
        }
        for value in values {
            sketch
                .add(value)
                .map_err(|error| Error::application(error.to_string()))?;
        }
        Ok(())
    }

    fn sketch_count(&mut self, sketch: &mut DDSketch) -> f64 {
        sketch.count()
    }

    fn sketch_encode(&mut self, sketch: &mut DDSketch) -> Vec<u8> {
        sketch.clone().encode_to_vec()
    }
}

fn validate_point(value: f64) -> Result<()> {
    if value.is_finite() && value >= 0.0 {
        Ok(())
    } else {
        Err(Error::argument(
            "sketch points must be finite and nonnegative",
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cocoon_gen::API;

    #[test]
    fn obfuscates_sql_and_rejects_malformed_traces() {
        let mut shim = Shim::default();
        let output = shim
            .obfuscate_sql("SELECT * FROM users WHERE id = 42".into())
            .unwrap();
        assert!(!output.contains("42"));
        assert!(output.contains('?'));
        assert_eq!(shim.obfuscate_traces(vec![0x90]).unwrap(), vec![0x90]);
        assert!(shim.obfuscate_traces(vec![0x90, 0]).is_err());
        assert!(shim.obfuscate_traces(Vec::new()).is_err());
    }

    #[test]
    fn sketch_batch_errors_are_atomic() {
        let mut shim = Shim::default();
        let mut sketch = shim.sketch_new();
        shim.sketch_add(&mut sketch, 1.0).unwrap();
        for invalid in [-1.0, f64::NAN, f64::INFINITY] {
            assert!(
                shim.sketch_add_many(&mut sketch, vec![2.0, invalid])
                    .is_err()
            );
            assert_eq!(shim.sketch_count(&mut sketch), 1.0);
        }
        shim.sketch_add_many(&mut sketch, vec![0.0, 2.0, 3.0])
            .unwrap();
        let decoded = DDSketch::from_encoded(&shim.sketch_encode(&mut sketch)).unwrap();
        assert_eq!(decoded.count(), 4.0);
    }
}
