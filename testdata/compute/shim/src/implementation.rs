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
        assert_eq!(shim.join("a".into(), "b".into()), "ab");
    }
}
