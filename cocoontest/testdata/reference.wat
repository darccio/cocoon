(module
  (memory (export "memory") 1 1)
  (data (i32.const 0) "\40\00\00\00\08\00\00\00")
  (func (export "cocoon_in_reserve") (param i32) (result i32) (i32.const 16))
  (func (export "cocoon_out") (result i32) (i32.const 0))
  (func (export "store") (param i64) (result i32)
    (i64.store (i32.const 64) (local.get 0))
    (i32.const 0))
)
