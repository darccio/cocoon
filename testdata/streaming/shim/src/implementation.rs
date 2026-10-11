use cocoon_guest::{Error, Result};

const WINDOW: usize = 1024;
const MAX_INPUT: usize = 65536;
const PATTERN: &[u8] = b"</head>";
const SNIPPET: &[u8] = b"<cocoon>";

#[derive(Default)]
pub struct Shim;

// All resource state is owned; no response body or borrowed snippet is retained.
pub struct Stream {
    pending: [u8; WINDOW],
    used: usize,
    ended: bool,
    injected: bool,
}

impl Stream {
    fn new() -> Self {
        Self {
            pending: [0; WINDOW],
            used: 0,
            ended: false,
            injected: false,
        }
    }

    // A match may start in the emittable prefix and finish in the held tail.
    // Consume the whole match with lookahead, so its bytes are never split into
    // an emitted prefix and a tail that could no longer be recognized.
    fn emit(&mut self, input: &[u8], prefix: usize) -> (Vec<u8>, usize) {
        let mut output = Vec::with_capacity(prefix + SNIPPET.len() + PATTERN.len());
        let mut consumed = 0;
        while consumed < prefix {
            if !self.injected && input[consumed..].starts_with(PATTERN) {
                output.extend_from_slice(SNIPPET);
                output.extend_from_slice(PATTERN);
                consumed += PATTERN.len();
                self.injected = true;
            } else {
                output.push(input[consumed]);
                consumed += 1;
            }
        }
        (output, consumed)
    }

    fn write(&mut self, chunk: Vec<u8>) -> Result<Vec<u8>> {
        if self.ended {
            return Err(Error::argument("stream has ended"));
        }
        if chunk.len() > MAX_INPUT {
            return Err(Error::limit());
        }
        let mut input = Vec::with_capacity(self.used + chunk.len());
        input.extend_from_slice(&self.pending[..self.used]);
        input.extend_from_slice(&chunk);
        let prefix = input.len().saturating_sub(WINDOW);
        let (output, consumed) = self.emit(&input, prefix);
        let tail = &input[consumed..];
        self.pending[..tail.len()].copy_from_slice(tail);
        self.used = tail.len();
        Ok(output)
    }

    fn end(&mut self) -> Result<Vec<u8>> {
        if self.ended {
            return Err(Error::argument("stream has ended"));
        }
        let pending = self.pending;
        let (output, _) = self.emit(&pending[..self.used], self.used);
        self.used = 0;
        self.ended = true;
        Ok(output)
    }
}

impl crate::cocoon_gen::API for Shim {
    type Stream = Stream;

    fn echo(&mut self, input: Vec<u8>) -> Vec<u8> {
        input
    }

    fn stream_new(&mut self) -> Stream {
        Stream::new()
    }

    fn stream_write(&mut self, stream: &mut Stream, chunk: Vec<u8>) -> Result<Vec<u8>> {
        stream.write(chunk)
    }

    fn stream_end(&mut self, stream: &mut Stream) -> Result<Vec<u8>> {
        stream.end()
    }

    fn stream_fault(&mut self, _stream: &mut Stream, kind: u32) -> Result<()> {
        match kind {
            0 => Ok(()),
            1 => Err(Error::application("deliberate streaming error")),
            2 => panic!("deliberate streaming trap"),
            _ => Err(Error::argument("unknown fault kind")),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cocoon_gen::API;

    // Independent whole-body oracle: splice once, without the streaming emit
    // loop, tail bookkeeping, or emission cutoff.
    fn oracle(input: &[u8]) -> Vec<u8> {
        let Some(index) = input
            .windows(PATTERN.len())
            .position(|part| part == PATTERN)
        else {
            return input.to_vec();
        };
        let mut output = input[..index].to_vec();
        output.extend_from_slice(SNIPPET);
        output.extend_from_slice(&input[index..]);
        output
    }

    fn check(input: &[u8], widths: &[usize]) {
        let mut shim = Shim;
        let mut stream = shim.stream_new();
        let mut output = Vec::new();
        let mut offset = 0;
        for &width in widths {
            let limit = input.len().min(offset + width);
            output.extend(
                shim.stream_write(&mut stream, input[offset..limit].to_vec())
                    .unwrap(),
            );
            assert!(stream.used <= WINDOW);
            offset = limit;
        }
        if offset < input.len() {
            output.extend(
                shim.stream_write(&mut stream, input[offset..].to_vec())
                    .unwrap(),
            );
            assert!(stream.used <= WINDOW);
        }
        output.extend(shim.stream_end(&mut stream).unwrap());
        assert_eq!(output, oracle(input));
        assert!(stream.ended);
        assert_eq!(stream.used, 0);
        assert_eq!(
            shim.stream_write(&mut stream, Vec::new())
                .unwrap_err()
                .status,
            cocoon_guest::ERR_ARG
        );
        assert_eq!(
            shim.stream_end(&mut stream).unwrap_err().status,
            cocoon_guest::ERR_ARG
        );
        shim.stream_close(stream);
    }

    #[test]
    fn every_pattern_split_and_byte_chunks() {
        for split in 0..=PATTERN.len() {
            let input = [b"prefix".as_slice(), PATTERN, b"suffix"].concat();
            check(&input, &[6 + split, 0, PATTERN.len() - split, 6]);
        }
        let input = [b"prefix".as_slice(), PATTERN, b"suffix"].concat();
        check(&input, &vec![1; input.len()]);
    }

    #[test]
    fn matches_cross_emission_boundaries() {
        for cutoff in 0..=PATTERN.len() {
            // The emittable prefix ends exactly at the match start, inside the
            // match, or immediately after it. Lookahead must consume full matches.
            let input = [
                vec![b'x'; 31],
                PATTERN.to_vec(),
                vec![b'y'; WINDOW - PATTERN.len() + cutoff],
            ]
            .concat();
            check(&input, &[input.len()]);
        }
        for prefix in WINDOW - PATTERN.len()..=WINDOW + PATTERN.len() {
            let input = [
                vec![b'x'; prefix],
                PATTERN.to_vec(),
                vec![b'y'; WINDOW + 11],
            ]
            .concat();
            for split in 0..=PATTERN.len() {
                check(&input, &[prefix + split, WINDOW, 1]);
            }
        }
    }

    #[test]
    fn absent_incomplete_repeated_overlapping_and_binary() {
        for input in [
            Vec::new(),
            b"ordinary body".to_vec(),
            b"</hea".to_vec(),
            b"</head></head>".to_vec(),
            b"<</head></hea</head>".to_vec(),
            [vec![0, 255, 128], PATTERN.to_vec(), vec![255, 0]].concat(),
            vec![b'x'; WINDOW],
            vec![b'x'; WINDOW + 1],
            vec![b'x'; MAX_INPUT],
        ] {
            check(&input, &[0, 1, 3, 1023, 4096]);
        }
        for length in 0..PATTERN.len() {
            check(&PATTERN[..length], &[1; 7]);
        }
        let input = [vec![b'x'; MAX_INPUT - PATTERN.len()], PATTERN.to_vec()].concat();
        check(&input, &[MAX_INPUT]);
    }

    #[test]
    fn bounded_state_and_owned_emitted_storage() {
        let mut stream = Stream::new();
        assert!(stream.write(vec![b'a'; WINDOW]).unwrap().is_empty());
        assert_eq!(stream.used, WINDOW);
        let first = stream.write(vec![b'b'; WINDOW]).unwrap();
        assert_eq!(first, vec![b'a'; WINDOW]);
        assert_eq!(stream.used, WINDOW);
        let second = stream.write(vec![b'c'; WINDOW]).unwrap();
        assert_eq!(first, vec![b'a'; WINDOW]);
        assert_eq!(second, vec![b'b'; WINDOW]);
        assert_eq!(stream.end().unwrap(), vec![b'c'; WINDOW]);
    }

    #[test]
    fn limits_and_expected_errors_preserve_state() {
        let mut shim = Shim;
        let mut stream = shim.stream_new();
        assert!(
            shim.stream_write(&mut stream, b"prefix</hea".to_vec())
                .unwrap()
                .is_empty()
        );
        let pending = stream.pending;
        let used = stream.used;
        assert_eq!(
            shim.stream_write(&mut stream, vec![0; MAX_INPUT + 1])
                .unwrap_err()
                .status,
            cocoon_guest::ERR_LIMIT
        );
        assert_eq!(stream.pending, pending);
        assert_eq!(stream.used, used);
        assert_eq!(
            shim.stream_fault(&mut stream, 1).unwrap_err().status,
            cocoon_guest::ERR_APP
        );
        assert_eq!(
            shim.stream_fault(&mut stream, 3).unwrap_err().status,
            cocoon_guest::ERR_ARG
        );
        assert!(shim.stream_fault(&mut stream, 0).is_ok());
        shim.stream_write(&mut stream, b"d>suffix".to_vec())
            .unwrap();
        assert_eq!(
            shim.stream_end(&mut stream).unwrap(),
            b"prefix<cocoon></head>suffix"
        );
        let empty = stream.pending;
        assert!(shim.stream_write(&mut stream, b"ignored".to_vec()).is_err());
        assert_eq!(stream.pending, empty);
        assert_eq!(stream.used, 0);
        assert!(stream.ended);
    }

    #[test]
    #[should_panic(expected = "deliberate streaming trap")]
    fn safe_native_trap() {
        let mut shim = Shim;
        let mut stream = shim.stream_new();
        let _ = shim.stream_fault(&mut stream, 2);
    }
}
