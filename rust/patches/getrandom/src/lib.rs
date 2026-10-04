//! Narrow, independently authored adapter for getrandom 0.2 users in Cocoon.
//! Cargo's additive `js` feature is intentionally accepted without JavaScript:
//! entropy always comes from the declared Cocoon host capability on wasm32.
//! The Unix backend exists for native shim tests; other native targets reject.

use std::{fmt, mem::MaybeUninit, num::NonZeroU32};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Error(NonZeroU32);

impl Error {
    pub const INTERNAL_START: u32 = 1 << 31;
    pub const CUSTOM_START: u32 = (1 << 31) + (1 << 30);
    pub const UNSUPPORTED: Self = Self(NonZeroU32::new(Self::INTERNAL_START).unwrap());
    pub const UNEXPECTED: Self = Self(NonZeroU32::new(Self::CUSTOM_START).unwrap());

    pub const fn code(self) -> NonZeroU32 {
        self.0
    }
    pub fn raw_os_error(self) -> Option<i32> {
        if self.0.get() < Self::INTERNAL_START {
            i32::try_from(self.0.get()).ok()
        } else {
            None
        }
    }
}

impl From<NonZeroU32> for Error {
    fn from(code: NonZeroU32) -> Self {
        Self(code)
    }
}
impl fmt::Display for Error {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(formatter, "Cocoon entropy failure (code {})", self.0)
    }
}
impl std::error::Error for Error {}
impl From<Error> for std::io::Error {
    fn from(error: Error) -> Self {
        match error.raw_os_error() {
            Some(code) => Self::from_raw_os_error(code),
            None => Self::other(error),
        }
    }
}

pub fn getrandom(data: &mut [u8]) -> Result<(), Error> {
    if data.is_empty() {
        return Ok(());
    }
    fill(data)
}

#[cfg(target_arch = "wasm32")]
fn fill(data: &mut [u8]) -> Result<(), Error> {
    cocoon_guest::random(data).map_err(|_| Error::UNEXPECTED)
}

#[cfg(all(not(target_arch = "wasm32"), unix))]
fn fill(data: &mut [u8]) -> Result<(), Error> {
    use std::io::Read;
    std::fs::File::open("/dev/urandom")
        .and_then(|mut file| file.read_exact(data))
        .map_err(|error| {
            error
                .raw_os_error()
                .and_then(|code| u32::try_from(code).ok())
                .and_then(NonZeroU32::new)
                .map(Error::from)
                .unwrap_or(Error::UNEXPECTED)
        })
}

#[cfg(all(not(target_arch = "wasm32"), not(unix)))]
fn fill(_data: &mut [u8]) -> Result<(), Error> {
    Err(Error::UNSUPPORTED)
}

pub fn getrandom_uninit(data: &mut [MaybeUninit<u8>]) -> Result<&mut [u8], Error> {
    for byte in data.iter_mut() {
        byte.write(0);
    }
    // SAFETY: every element is initialized above, with identical size/alignment.
    // The returned mutable view is tied to the exclusive input slice's lifetime.
    let initialized =
        unsafe { std::slice::from_raw_parts_mut(data.as_mut_ptr().cast(), data.len()) };
    getrandom(initialized)?;
    Ok(initialized)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn empty_and_error_contracts() {
        getrandom(&mut []).unwrap();
        let error = Error::from(NonZeroU32::new(42).unwrap());
        assert_eq!(error.code().get(), 42);
        assert_eq!(error.raw_os_error(), Some(42));
        assert_eq!(Error::UNSUPPORTED.raw_os_error(), None);
        assert!(error.to_string().contains("42"));
        assert_eq!(std::io::Error::from(error).raw_os_error(), Some(42));
        assert!(
            std::io::Error::from(Error::UNEXPECTED)
                .to_string()
                .contains("entropy")
        );
    }

    #[cfg(unix)]
    #[test]
    fn initializes_every_byte() {
        let mut data = [MaybeUninit::uninit(); 64];
        let initialized = getrandom_uninit(&mut data).unwrap();
        assert_eq!(initialized.len(), 64);
        assert!(initialized.iter().any(|byte| *byte != 0));
    }
}
