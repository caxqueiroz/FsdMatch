# sample-spring-app

A synthetic Spring Boot fixture authored for fsdtrace's tests. None of
this code is derived from any real project. It deliberately exercises
each Spring annotation kind listed in SPEC §7.2 so the harvester has at
least one row per kind in its output.

The structure is `pom`-less and only the source layout is real — there
is no expectation that this compiles. fsdtrace's tree-sitter pass reads
the files directly.
