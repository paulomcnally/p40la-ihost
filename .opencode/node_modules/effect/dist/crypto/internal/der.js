/**
 * The few DER operations `KeyFormat` needs to turn PKCS#1 and SEC1 private
 * keys into PKCS#8, and to find the public point of an EC private key.
 */
/**
 * Reads the element at `offset`, returning `undefined` when it does not fit
 * in `end` or its length is not in definite form of at most 3 bytes.
 *
 * @internal
 */
export const read = (bytes, offset, end = bytes.length) => {
  if (offset + 2 > end) return undefined;
  const tag = bytes[offset];
  const first = bytes[offset + 1];
  let start = offset + 2;
  let length = first;
  if (first >= 0x80) {
    const size = first & 0x7f;
    if (size === 0 || size > 3 || start + size > end) return undefined;
    length = 0;
    for (let i = 0; i < size; i++) length = length * 256 + bytes[start++];
  }
  return start + length > end ? undefined : {
    tag,
    offset,
    start,
    end: start + length
  };
};
/**
 * Reads a whole input as one element, with nothing after it.
 *
 * @internal
 */
export const readOnly = (bytes, tag) => {
  const element = read(bytes, 0);
  return element !== undefined && element.tag === tag && element.end === bytes.length ? element : undefined;
};
/**
 * Reads the elements inside a constructed element, returning `undefined`
 * when they do not fill it exactly.
 *
 * @internal
 */
export const children = (bytes, parent) => {
  const out = [];
  let offset = parent.start;
  while (offset < parent.end) {
    const element = read(bytes, offset, parent.end);
    if (element === undefined) return undefined;
    out.push(element);
    offset = element.end;
  }
  return out;
};
/** @internal */
export const slice = (bytes, element) => bytes.subarray(element.offset, element.end);
const length = n => n < 0x80 ? [n] : n < 0x100 ? [0x81, n] : n < 0x10000 ? [0x82, n >> 8, n & 0xff] : [0x83, n >> 16, n >> 8 & 0xff, n & 0xff];
/**
 * Encodes an element from its tag and contents.
 *
 * @internal
 */
export const encode = (tag, ...contents) => {
  const size = contents.reduce((sum, part) => sum + part.length, 0);
  const header = [tag, ...length(size)];
  const out = new Uint8Array(header.length + size);
  out.set(header);
  let offset = header.length;
  for (const part of contents) {
    out.set(part, offset);
    offset += part.length;
  }
  return out;
};
/** @internal */
export const sequence = 0x30;
/** @internal */
export const octetString = 0x04;
/** @internal */
export const integer = 0x02;
/** @internal */
export const objectIdentifier = 0x06;
/** @internal */
export const equals = (a, b) => a.length === b.length && a.every((byte, i) => byte === b[i]);
//# sourceMappingURL=der.js.map