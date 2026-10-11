#[derive(Default)]
pub struct Shim;
impl crate::cocoon_gen::API for Shim {
    type Counter = u64;

    fn echo(&mut self, input: String) -> cocoon_guest::Result<String> {
        Ok(input)
    }
    fn join(&mut self, left: String, right: String) -> String {
        left + &right
    }
    fn bounce(&mut self, payload: crate::cocoon_gen::Payload) -> crate::cocoon_gen::Payload {
        payload
    }
    fn bounce_slices(
        &mut self,
        label: String,
        signed32: Vec<i32>,
        data: Vec<u8>,
        unsigned32: Vec<u32>,
        numbers: crate::cocoon_gen::Numbers,
        signed64: Vec<i64>,
        unsigned64: Vec<u64>,
        float32: Vec<f32>,
        float64: Vec<f64>,
    ) -> crate::cocoon_gen::Payload {
        crate::cocoon_gen::Payload {
            label,
            data,
            numbers,
            signed32,
            unsigned32,
            signed64,
            unsigned64,
            float32,
            float64,
        }
    }
    fn scalar_values(
        &mut self,
        a: i32,
        b: u32,
        c: i64,
        d: u64,
        e: f32,
        f: f64,
        enabled: bool,
    ) -> crate::cocoon_gen::Numbers {
        crate::cocoon_gen::Numbers {
            signed32: a,
            unsigned32: b,
            signed64: c,
            unsigned64: d,
            float32: e,
            float64: f,
            enabled,
        }
    }
    fn test_trap(&mut self, kind: u32) {
        assert_eq!(kind, 0, "deliberate safe Rust trap");
    }
    fn counter_new(&mut self, start: u64) -> u64 {
        start
    }
    fn counter_add(&mut self, counter: &mut u64, delta: u64) -> cocoon_guest::Result<u64> {
        *counter = counter
            .checked_add(delta)
            .ok_or_else(|| cocoon_guest::Error::application("counter overflow"))?;
        Ok(*counter)
    }
    fn counter_count(&mut self, counter: &mut u64) -> u64 {
        *counter
    }
    fn counter_sum(&mut self, counter: &mut u64, deltas: Vec<u64>) -> cocoon_guest::Result<u64> {
        // Validate the entire vector before committing any state change.
        let mut total = *counter;
        for delta in deltas {
            total = total
                .checked_add(delta)
                .ok_or_else(|| cocoon_guest::Error::application("counter overflow"))?;
        }
        *counter = total;
        Ok(total)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cocoon_gen::API;

    #[test]
    fn scalars_and_checked_resources() {
        let mut shim = Shim;
        let numbers = shim.scalar_values(-1, u32::MAX, i64::MIN, u64::MAX, -0.0, 1.25, true);
        assert_eq!(
            crate::cocoon_gen::Numbers::decode(&numbers.clone().encode().unwrap()).unwrap(),
            numbers
        );
        let mut counter = shim.counter_new(u64::MAX);
        assert!(shim.counter_add(&mut counter, 1).is_err());
        assert_eq!(shim.counter_count(&mut counter), u64::MAX);
        let mut counter = shim.counter_new(7);
        assert_eq!(shim.counter_sum(&mut counter, vec![2, 3]), Ok(12));
        assert_eq!(shim.counter_sum(&mut counter, vec![]), Ok(12));
        assert_eq!(
            shim.counter_sum(&mut counter, vec![1, u64::MAX]),
            Err(cocoon_guest::Error::application("counter overflow"))
        );
        assert_eq!(shim.counter_count(&mut counter), 12);
        assert_eq!(shim.join("a".into(), "b".into()), "ab");
    }
}
