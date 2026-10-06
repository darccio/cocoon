(module
  (memory 1 1)
  (func (export "floor_sat") (param f64) (result i32)
    (i32.trunc_sat_f64_s (f64.floor (local.get 0))))
  (func (export "f64_unfused") (param f64 f64 f64) (result i64)
    (i64.reinterpret_f64
      (f64.add (f64.mul (local.get 0) (local.get 1)) (local.get 2))))
  (func (export "f32_unfused") (param f32 f32 f32) (result i32)
    (i32.reinterpret_f32
      (f32.add (f32.mul (local.get 0) (local.get 1)) (local.get 2))))
  (func (export "rounding_order") (param f64) (result i64)
    (i64.reinterpret_f64
      (f64.add (f64.add (local.get 0) (f64.const 1e16)) (f64.const -1e16))))
  (func (export "subnormal_half") (param f64) (result i64)
    (i64.reinterpret_f64 (f64.mul (local.get 0) (f64.const 0.5))))
  (func $effect (param f64) (result f64)
    (i32.store (i32.const 0)
      (i32.add (i32.load (i32.const 0)) (i32.const 1)))
    (local.get 0))
  (func (export "floor_effect") (param f64) (result i32)
    (i32.trunc_sat_f64_s (f64.floor (call $effect (local.get 0)))))
  (func (export "counter") (result i32)
    (i32.load (i32.const 0)))
  (func (export "reset")
    (i32.store (i32.const 0) (i32.const 0)))
  (func (export "floor_trap") (param i32) (result i32)
    (local i32)
    (i32.store (i32.const 0) (i32.const 7))
    (local.set 1
      (i32.trunc_sat_f64_s (f64.floor (f64.load (local.get 0)))))
    (i32.store (i32.const 0) (i32.const 9))
    (local.get 1))
  (func (export "effect_then_trap") (param i32 f64) (result i32)
    (local i32)
    (local.set 2
      (i32.trunc_sat_f64_s
        (f64.floor
          (f64.add (call $effect (local.get 1)) (f64.load (local.get 0))))))
    (i32.store (i32.const 0) (i32.const 9))
    (local.get 2)))
