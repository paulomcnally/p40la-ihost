import * as Effect from "../../Effect.ts";
import type { StoreId } from "../EventLogMessage.ts";
/** Rules and audit logs shared by the memory and SQL compaction storages. */
export declare const hourMillis: number;
/** The newest storage-time hour that ended at least `lagMillis` ago. */
export declare const lagHour: (now: number, lagMillis: number) => number;
/** A staged batch's position packs its segment index and its index in the segment. */
export declare const positionsPerSegment: number;
export declare const maxSegments: number;
export declare const logCommit: (options: {
    readonly publicKey: string;
    readonly storeId: StoreId;
    readonly result: "Committed" | "Superseded";
    readonly through: number;
    readonly keptCount: number;
    readonly activationId: string | undefined;
}) => Effect.Effect<void>;
export declare const logNoOp: (options: {
    readonly publicKey: string;
    readonly storeId: StoreId;
    readonly stageId: string;
    readonly through: number;
}) => Effect.Effect<void>;
export declare const logRollback: (options: {
    readonly publicKey: string;
    readonly storeId: StoreId;
    readonly activation: {
        readonly id: string;
        readonly base: number;
        readonly keptCount: number | undefined;
    };
    readonly withdrawn: {
        readonly id: string;
        readonly base: number;
        readonly keptCount: number | undefined;
    };
}) => Effect.Effect<void>;
//# sourceMappingURL=compaction.d.ts.map