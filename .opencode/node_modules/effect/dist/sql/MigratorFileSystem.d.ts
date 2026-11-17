import { FileSystem } from "../FileSystem.ts";
import { Path } from "../Path.ts";
import { type Loader } from "./Migrator.ts";
/**
 * Creates a migration loader that reads a directory with `FileSystem`, imports
 * files named `<id>_<name>.js`, `<id>_<name>.ts`,
 * `<id>_<name>.mjs`, or `<id>_<name>.mts`, and sorts migrations by id.
 *
 * **Details**
 *
 * Requires a `Path` service appropriate for the migration directory's path
 * syntax. On Windows, prefer a platform-aware implementation such as
 * `NodePath.layer`; the core `Path.layer` uses POSIX semantics and does not
 * preserve Windows drive-letter paths.
 *
 * @stability unstable
 * @category loaders
 * @since 4.0.0
 */
export declare const fromFileSystem: (directory: string) => Loader<FileSystem | Path>;
//# sourceMappingURL=MigratorFileSystem.d.ts.map