Cog Project Review — Biggest Problems
I've read the entire codebase. Here are the most significant issues, ordered by impact:

9. No source position tracking through transpilation
The /​/line directives appended in attachLineDecl provide some mapping, but they're text comments appended after AST construction. There's no structured source map. When the user gets a Go compile error, they'll see Cog line numbers (good), but for runtime errors (stack traces), the mapping may be incomplete or misleading. This becomes critical for debugging.
