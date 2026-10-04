//! The authored Datadog implementation is safe Rust; pointer glue is generated.
mod cocoon_gen;
#[forbid(unsafe_code)]
mod implementation;
pub use implementation::Shim;
