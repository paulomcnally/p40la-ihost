/** The most entries a remote batch holds. */
export const maxBatchEntries = 100;
/** The most payload bytes a remote batch holds, unless it is a single entry. */
export const maxBatchPayloadBytes = 256 * 1024;
/**
 * Packs entries into batches of at most `maxBatchEntries` entries and
 * `maxBatchPayloadBytes` of payload. A larger entry is a batch on its own.
 */
export const toBatches = entries => {
  const batches = [];
  let current = [];
  let bytes = 0;
  for (const entry of entries) {
    const size = entry.payload.byteLength;
    if (current.length > 0 && (current.length === maxBatchEntries || bytes + size > maxBatchPayloadBytes)) {
      batches.push(current);
      current = [];
      bytes = 0;
    }
    current.push(entry);
    bytes += size;
  }
  if (current.length > 0) batches.push(current);
  return batches;
};
//# sourceMappingURL=batches.js.map