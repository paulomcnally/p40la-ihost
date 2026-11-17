import * as Effect from "../../Effect.ts";
import * as Stream from "../../Stream.ts";
import type { Entry } from "../EventJournal.ts";
import type { Registry } from "../EventLog.ts";
declare const IdempotenceError_base: new <A extends Record<string, any> = {}>(args: import("../../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../../Cause.ts").YieldableError & {
    readonly _tag: "IdempotenceError";
} & Readonly<A>;
/** A second pass over the kept set dropped `entries`, so a compactor does not compose. */
export declare class IdempotenceError extends IdempotenceError_base<{
    readonly entries: ReadonlyArray<Entry>;
}> {
}
/**
 * Folds the opted-in compactors over `entries`, which arrive in input order
 * without repeated ids, and returns the kept entries in input order.
 *
 * Each step folds a chunk into the kept set. The chunk starts at `chunkSize`
 * and grows to the size of the kept set, so a run stays linear when little is
 * dropped and memory stays at about twice the kept set. A compactor that
 * composes keeps the same entries at any chunking.
 *
 * Fails with `IdempotenceError` when a second pass over the kept set drops
 * more. A compactor defect fails the run.
 */
export declare const run: <E, R>(options: {
    readonly compactors: Registry["Service"]["compactors"];
    readonly entries: Stream.Stream<Entry, E, R>;
    readonly chunkSize: number;
}) => Effect.Effect<ReadonlyArray<Entry>, E | IdempotenceError, R>;
export {};
//# sourceMappingURL=remoteCompaction.d.ts.map