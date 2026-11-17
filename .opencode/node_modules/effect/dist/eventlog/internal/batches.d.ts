import type { NonEmptyArray } from "../../Array.ts";
import type { Entry } from "../EventJournal.ts";
/** The most entries a remote batch holds. */
export declare const maxBatchEntries = 100;
/** The most payload bytes a remote batch holds, unless it is a single entry. */
export declare const maxBatchPayloadBytes: number;
/**
 * Packs entries into batches of at most `maxBatchEntries` entries and
 * `maxBatchPayloadBytes` of payload. A larger entry is a batch on its own.
 */
export declare const toBatches: (entries: ReadonlyArray<Entry>) => Array<NonEmptyArray<Entry>>;
//# sourceMappingURL=batches.d.ts.map