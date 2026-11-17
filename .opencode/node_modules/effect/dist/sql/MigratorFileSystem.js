var __rewriteRelativeImportExtension = this && this.__rewriteRelativeImportExtension || function (path, preserveJsx) {
  if (typeof path === "string" && /^\.\.?\//.test(path)) {
    return path.replace(/\.(tsx)$|((?:\.d)?)((?:\.[^./]+?)?)\.([cm]?)ts$/i, function (m, tsx, d, ext, cm) {
      return tsx ? preserveJsx ? ".jsx" : ".js" : d && (!ext || !cm) ? m : d + ext + "." + cm.toLowerCase() + "js";
    });
  }
  return path;
};
/**
 * Loads SQL migrations by reading a directory.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Effect from "../Effect.js";
import { FileSystem } from "../FileSystem.js";
import * as Option from "../Option.js";
import { Path } from "../Path.js";
import { MigrationError } from "./Migrator.js";
// Kept out of `Migrator.ts` and the `effect/sql` barrel: Metro rejects the
// non-literal `import()` below in any module it bundles.
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
export const fromFileSystem = /*#__PURE__*/Effect.fnUntraced(function* (directory) {
  const Fs = yield* FileSystem;
  const path = yield* Path;
  const files = yield* Effect.mapError(Fs.readDirectory(directory), cause => new MigrationError({
    kind: "Failed",
    cause,
    message: "Failed to read migrations directory"
  }));
  return files.map(file => Option.fromNullishOr(file.match(/^(?:.*\/)?(\d+)_([^.]+)\.(js|ts|mjs|mts)$/))).flatMap(Option.match({
    onNone: () => [],
    onSome: ([basename, id, name]) => [[Number(id), name,
    // `import` needs a file URL: on Windows an absolute path such as
    // `D:\migrations\1_init.ts` is rejected by the ESM loader. `orDie` keeps the
    // failure a defect so `loadMigration` reports it as an import error.
    Effect.flatMap(Effect.orDie(path.toFileUrl(path.join(directory, basename))), url => Effect.promise(() => import(__rewriteRelativeImportExtension(/* @vite-ignore */
    /* webpackIgnore: true */
    url.href))))]]
  })).sort(([a], [b]) => a - b);
});
//# sourceMappingURL=MigratorFileSystem.js.map