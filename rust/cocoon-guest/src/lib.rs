//! Safe guest primitives for Cocoon ABI v3. Only generated pointer glue is unsafe.

use std::cell::RefCell;

pub const ABI_VERSION: u32 = 3;
pub const OK: i32 = 0;
pub const ERR_APP: i32 = 1;
pub const ERR_ARG: i32 = 2;
pub const ERR_HANDLE: i32 = 3;
pub const ERR_LIMIT: i32 = 5;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error {
    pub status: i32,
    pub message: String,
}

impl Error {
    pub fn new(status: i32, message: impl Into<String>) -> Self {
        Self {
            status,
            message: message.into(),
        }
    }
    pub fn argument(message: impl Into<String>) -> Self {
        Self::new(ERR_ARG, message)
    }
    pub fn application(message: impl Into<String>) -> Self {
        Self::new(ERR_APP, message)
    }
    pub fn limit() -> Self {
        Self::new(ERR_LIMIT, "size limit exceeded")
    }
    pub fn handle() -> Self {
        Self::new(ERR_HANDLE, "stale or unknown handle")
    }
}

pub type Result<T> = std::result::Result<T, Error>;

#[repr(C)]
#[derive(Default)]
pub struct Output {
    pub pointer: u32,
    pub length: u32,
}

#[derive(Default)]
struct Buffers {
    input: Vec<u8>,
    output: Vec<u8>,
    descriptor: Output,
    max_input: usize,
    max_output: usize,
}

thread_local! { static BUFFERS: RefCell<Buffers> = RefCell::new(Buffers::default()); }

pub fn init(max_input: usize, max_output: usize) {
    BUFFERS.with_borrow_mut(|b| {
        b.max_input = max_input;
        b.max_output = max_output;
    });
    #[cfg(all(target_arch = "wasm32", feature = "log"))]
    std::panic::set_hook(Box::new(|information| {
        let message = information.to_string();
        unsafe {
            host_log(4, message.as_ptr(), message.len());
        }
    }));
}

pub fn reserve(length: usize) -> *mut u8 {
    BUFFERS.with_borrow_mut(|b| {
        assert!(length <= b.max_input, "input limit exceeded");
        b.input.resize(length, 0);
        b.input.as_mut_ptr()
    })
}

/// Validates that input pointers belong to the current reserved input buffer.
pub fn input(pointer: usize, length: usize) -> Result<Vec<u8>> {
    BUFFERS.with_borrow(|b| {
        let base = b.input.as_ptr() as usize;
        let offset = pointer
            .checked_sub(base)
            .ok_or_else(|| Error::argument("foreign input pointer"))?;
        let end = offset
            .checked_add(length)
            .ok_or_else(|| Error::argument("input overflow"))?;
        let data = b
            .input
            .get(offset..end)
            .ok_or_else(|| Error::argument("input out of bounds"))?;
        Ok(data.to_vec())
    })
}

pub fn output() -> *const Output {
    BUFFERS.with_borrow(|b| &b.descriptor as *const Output)
}

pub fn reply(result: Result<Vec<u8>>) -> i32 {
    BUFFERS.with_borrow_mut(|b| {
        let (mut status, data) = match result {
            Ok(data) => (OK, data),
            Err(e) => (e.status, e.message.into_bytes()),
        };
        b.output.clear();
        if data.len() > b.max_output {
            status = ERR_LIMIT;
        } else {
            b.output.extend_from_slice(&data);
        }
        b.descriptor.pointer = b.output.as_ptr() as usize as u32;
        b.descriptor.length = b.output.len() as u32;
        status
    })
}

/// Publishes a canonical empty success reply without constructing a byte vector.
#[inline(always)]
pub fn reply_unit(result: Result<()>) -> i32 {
    match result {
        Ok(()) => BUFFERS.with_borrow_mut(|b| {
            b.output.clear();
            b.descriptor.pointer = b.output.as_ptr() as usize as u32;
            b.descriptor.length = 0;
            OK
        }),
        Err(error) => reply_unit_error(error),
    }
}

#[cold]
fn reply_unit_error(error: Error) -> i32 {
    reply(Err(error))
}

pub fn trim(keep: usize) {
    BUFFERS.with_borrow_mut(|b| {
        if b.input.capacity() > keep {
            b.input = Vec::new();
        }
        if b.output.capacity() > keep {
            b.output = Vec::new();
        }
        b.descriptor = Output::default();
    });
}

pub fn alloc(length: usize) -> *mut u8 {
    reserve(length)
}

#[cfg(all(target_arch = "wasm32", feature = "log"))]
#[link(wasm_import_module = "cocoon")]
unsafe extern "C" {
    #[link_name = "log"]
    fn host_log(level: i32, pointer: *const u8, length: usize);
}

#[cfg(all(target_arch = "wasm32", feature = "random"))]
#[link(wasm_import_module = "cocoon")]
unsafe extern "C" {
    #[link_name = "random_get"]
    fn host_random(pointer: *mut u8, length: usize) -> i32;
}

#[cfg(all(target_arch = "wasm32", feature = "clock"))]
#[link(wasm_import_module = "cocoon")]
unsafe extern "C" {
    #[link_name = "clock_nanos"]
    fn host_clock() -> i64;
}

#[cfg(all(target_arch = "wasm32", feature = "random"))]
pub fn random(data: &mut [u8]) -> Result<()> {
    if unsafe { host_random(data.as_mut_ptr(), data.len()) } == 0 {
        Ok(())
    } else {
        Err(Error::application("host randomness failed"))
    }
}

#[cfg(all(target_arch = "wasm32", feature = "clock"))]
pub fn clock_nanos() -> i64 {
    unsafe { host_clock() }
}

struct Slot<T> {
    value: Option<T>,
    generation: u32,
}

/// Generation-tagged handles never resurrect a removed resource, even on wraparound.
pub struct Slab<T> {
    slots: Vec<Slot<T>>,
    free: Vec<u32>,
}

impl<T> Default for Slab<T> {
    fn default() -> Self {
        Self {
            slots: Vec::new(),
            free: Vec::new(),
        }
    }
}

impl<T> Slab<T> {
    pub fn insert(&mut self, value: T) -> Result<u64> {
        let index = match self.free.pop() {
            Some(index) => index,
            None => {
                let index = u32::try_from(self.slots.len()).map_err(|_| Error::limit())?;
                if index == u32::MAX {
                    return Err(Error::limit());
                }
                self.slots.push(Slot {
                    value: None,
                    generation: 1,
                });
                index
            }
        };
        let slot = &mut self.slots[index as usize];
        slot.value = Some(value);
        Ok((u64::from(slot.generation) << 32) | (u64::from(index) + 1))
    }
    fn index(&self, handle: u64) -> Result<usize> {
        let index = (handle as u32).checked_sub(1).ok_or_else(Error::handle)? as usize;
        let slot = self.slots.get(index).ok_or_else(Error::handle)?;
        if slot.generation != (handle >> 32) as u32 || slot.value.is_none() {
            return Err(Error::handle());
        }
        Ok(index)
    }
    pub fn get_mut(&mut self, handle: u64) -> Result<&mut T> {
        let index = self.index(handle)?;
        self.slots[index].value.as_mut().ok_or_else(Error::handle)
    }
    pub fn remove(&mut self, handle: u64) -> Result<T> {
        let index = self.index(handle)?;
        let slot = &mut self.slots[index];
        let value = slot.value.take().ok_or_else(Error::handle)?;
        if let Some(next) = slot.generation.checked_add(1) {
            slot.generation = next;
            self.free.push(index as u32);
        }
        Ok(value)
    }
}

pub fn record_size(fields: &[Vec<u8>], maximum: usize) -> Result<usize> {
    let mut size = fields
        .len()
        .checked_mul(8)
        .and_then(|n| n.checked_add(4))
        .ok_or_else(Error::limit)?;
    for field in fields {
        size = size.checked_add(field.len()).ok_or_else(Error::limit)?;
    }
    if size > maximum || size > u32::MAX as usize {
        return Err(Error::limit());
    }
    Ok(size)
}

pub fn encode_record(fields: &[Vec<u8>], maximum: usize) -> Result<Vec<u8>> {
    let size = record_size(fields, maximum)?;
    let mut data = Vec::with_capacity(size);
    data.extend_from_slice(&(fields.len() as u32).to_le_bytes());
    for (index, field) in fields.iter().enumerate() {
        data.extend_from_slice(&((index + 1) as u32).to_le_bytes());
        data.extend_from_slice(&(field.len() as u32).to_le_bytes());
        data.extend_from_slice(field);
    }
    Ok(data)
}

pub fn decode_record(data: &[u8], expected: usize, maximum: usize) -> Result<Vec<&[u8]>> {
    if data.len() > maximum {
        return Err(Error::limit());
    }
    if data.len() < 4
        || expected > (data.len() - 4) / 8
        || u32::from_le_bytes(data[..4].try_into().map_err(|_| Error::argument("count"))?) as usize
            != expected
    {
        return Err(Error::argument("field count"));
    }
    let mut rest = &data[4..];
    let mut fields = Vec::with_capacity(expected);
    for index in 0..expected {
        if rest.len() < 8 {
            return Err(Error::argument("truncated field"));
        }
        let id = u32::from_le_bytes(
            rest[..4]
                .try_into()
                .map_err(|_| Error::argument("field id"))?,
        );
        let size = u32::from_le_bytes(
            rest[4..8]
                .try_into()
                .map_err(|_| Error::argument("field length"))?,
        ) as usize;
        if id as usize != index + 1 {
            return Err(Error::argument("field order"));
        }
        rest = &rest[8..];
        if size > rest.len() {
            return Err(Error::argument("truncated payload"));
        }
        fields.push(&rest[..size]);
        rest = &rest[size..];
    }
    if !rest.is_empty() {
        return Err(Error::argument("trailing data"));
    }
    Ok(fields)
}

pub fn decode_f64s(data: &[u8]) -> Result<Vec<f64>> {
    if !data.len().is_multiple_of(8) {
        return Err(Error::argument("invalid scalar slice length"));
    }
    data.chunks_exact(8)
        .map(|chunk| {
            Ok(f64::from_le_bytes(
                chunk.try_into().map_err(|_| Error::argument("float"))?,
            ))
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn slab_generation_and_isolation() {
        let mut slab = Slab::default();
        let first = slab.insert(42).unwrap();
        assert_eq!(slab.remove(first).unwrap(), 42);
        assert!(slab.get_mut(first).is_err());
        let second = slab.insert(43).unwrap();
        assert_ne!(first, second);
        assert_eq!(*slab.get_mut(second).unwrap(), 43);
        assert!(slab.get_mut(0).is_err());
        assert!(slab.get_mut(u64::MAX).is_err());
        slab.slots[0].generation = u32::MAX;
        let last = (u64::from(u32::MAX) << 32) | 1;
        assert_eq!(slab.remove(last).unwrap(), 43);
        assert_eq!(slab.insert(44).unwrap() as u32, 2);
    }
    #[test]
    fn slab_mutable_lookup_preserves_handle_checks_and_reuse() {
        let mut slab = Slab::default();
        let first = (1_u64 << 32) | 1;
        assert_eq!(slab.get_mut(first).unwrap_err(), Error::handle());
        assert_eq!(slab.insert(42).unwrap(), first);
        for invalid in [0, 1_u64 << 32, 1, (2_u64 << 32) | 1, first + 1, u64::MAX] {
            assert_eq!(
                slab.get_mut(invalid).unwrap_err(),
                Error::handle(),
                "handle={invalid:016x}"
            );
        }
        *slab.get_mut(first).unwrap() = 43;
        assert_eq!(slab.remove(first).unwrap(), 43);
        let second = (2_u64 << 32) | 1;
        // The generation matches, but the removed slot is empty.
        assert_eq!(slab.get_mut(second).unwrap_err(), Error::handle());
        assert_eq!(slab.get_mut(first).unwrap_err(), Error::handle());
        assert_eq!(slab.insert(44).unwrap(), second);
        assert_eq!(*slab.get_mut(second).unwrap(), 44);

        slab.slots[0].generation = u32::MAX;
        let last = (u64::from(u32::MAX) << 32) | 1;
        *slab.get_mut(last).unwrap() = 45;
        assert_eq!(slab.remove(last).unwrap(), 45);
        // A retired slot stays empty even when the handle generation matches.
        assert_eq!(slab.get_mut(last).unwrap_err(), Error::handle());
        let next = slab.insert(46).unwrap();
        assert_eq!(next, (1_u64 << 32) | 2);
        assert_eq!(*slab.get_mut(next).unwrap(), 46);
        assert_eq!(slab.get_mut(last).unwrap_err(), Error::handle());
    }
    #[test]
    fn canonical_record() {
        let fields = vec![b"a=b\nvalue".to_vec(), vec![0, 255]];
        let data = encode_record(&fields, 128).unwrap();
        assert_eq!(decode_record(&data, 2, 128).unwrap(), fields);
        for end in 0..data.len() {
            assert!(decode_record(&data[..end], 2, 128).is_err());
        }
        let mut bad = data.clone();
        bad[4] = 2;
        assert!(decode_record(&bad, 2, 128).is_err());
        let mut bad = data.clone();
        bad.push(0);
        assert!(decode_record(&bad, 2, 128).is_err());
        assert!(encode_record(&fields, 1).is_err());
        assert!(decode_record(&data, 2, 1).is_err());
        assert!(decode_record(&[0, 0, 0, 0], usize::MAX, 128).is_err());
        assert_eq!(encode_record(&[], 4).unwrap(), [0, 0, 0, 0]);
    }
    #[test]
    fn input_reply_and_limits() {
        init(64, 4);
        let pointer = reserve(8) as usize;
        assert_eq!(input(pointer, 8).unwrap(), vec![0; 8]);
        assert!(input(pointer, 9).is_err());
        assert!(input(usize::MAX, 1).is_err());
        assert_eq!(reply(Ok(vec![1, 2])), OK);
        assert_eq!(reply(Ok(vec![0; 5])), ERR_LIMIT);
        assert_eq!(reply(Err(Error::new(ERR_APP, "err"))), ERR_APP);
        trim(0);
        assert_eq!(BUFFERS.with_borrow(|b| b.descriptor.length), 0);
    }
    #[test]
    fn unit_reply_clears_previous_output_and_preserves_errors() {
        init(64, 32);
        let descriptor = output();
        for previous in [Ok(vec![1, 2, 3]), Err(Error::application("previous error"))] {
            reply(previous);
            assert!(BUFFERS.with_borrow(|b| b.descriptor.length > 0));
            assert_eq!(reply_unit(Ok(())), OK);
            assert_eq!(output(), descriptor);
            BUFFERS.with_borrow(|b| {
                assert!(b.output.is_empty());
                assert_eq!(b.descriptor.length, 0);
                assert_eq!(b.descriptor.pointer, b.output.as_ptr() as usize as u32);
            });
        }
        assert_eq!(reply_unit(Err(Error::argument("bad"))), ERR_ARG);
        BUFFERS.with_borrow(|b| {
            assert_eq!(b.output, b"bad");
            assert_eq!(b.descriptor.length, 3);
        });
        init(64, 2);
        assert_eq!(reply_unit(Err(Error::application("too long"))), ERR_LIMIT);
        assert_eq!(BUFFERS.with_borrow(|b| b.descriptor.length), 0);
        trim(0);
        init(64, 0);
        assert_eq!(reply_unit(Ok(())), OK);
        assert_eq!(output(), descriptor);
        assert_eq!(BUFFERS.with_borrow(|b| b.descriptor.length), 0);
    }
    #[test]
    fn unit_reply_preserves_every_error_status() {
        init(64, 128);
        let descriptor = output();
        for error in [
            Error::argument("bad\0世界"),
            Error::application("application error"),
            Error::limit(),
            Error::handle(),
        ] {
            let status = error.status;
            let message = error.message.clone();
            assert_eq!(reply_unit(Err(error)), status);
            assert_eq!(output(), descriptor);
            BUFFERS.with_borrow(|b| {
                assert_eq!(b.output, message.as_bytes());
                assert_eq!(b.descriptor.length as usize, message.len());
            });
            assert_eq!(reply_unit(Ok(())), OK);
            assert_eq!(output(), descriptor);
            BUFFERS.with_borrow(|b| {
                assert!(b.output.is_empty());
                assert_eq!(b.descriptor.length, 0);
            });
        }
    }
    #[test]
    fn scalar_slice() {
        assert_eq!(decode_f64s(&1.5_f64.to_le_bytes()).unwrap(), [1.5]);
        assert!(decode_f64s(&[0]).is_err());
    }
}
