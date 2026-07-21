# auto-fix ケース仕様

本書は `testdata/fix/<case>` と同期する auto-fix ユースケース仕様である。新しい fix ケースを追加するときは、まず本書にケース名を `testdata/fix/<case>` のディレクトリ名と完全一致させて追記し、その後に testdata と実装を追加する。

## rename_to_majority

- **目的**: 単一ファイル内で同一 import path の canonical alias が多数決で定まったとき、既存の無 alias import と qualified identifier を canonical alias へ rename できることを確認する。
- **入力の要点**: `fmt` が無 alias で import され、`fmt.Println` が使われている。
- **期待される出力**: import が `f "fmt"` になり、参照が `f.Println` に書き換わる。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## multi_file_majority

- **目的**: 複数ファイルをまたいだ alias 多数決により、少数派の alias だけを canonical alias へ統合できることを確認する。
- **入力の要点**: `foo.go` と `bar.go` は `f "fmt"` を使い、`boo.go` だけが `oldf "fmt"` と `oldf.Println` を使っている。
- **期待される出力**: `boo.go` の import と参照だけが `f` に書き換わり、すでに canonical alias を使う `foo.go` と `bar.go` は変わらない。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## collision_unaliased_to_alias

- **目的**: 無 alias import を alias 付き import へ rename するとローカル識別子と衝突する場合に、危険な auto-fix をスキップできることを確認する。
- **入力の要点**: `fmt` が無 alias で import され、同じ関数スコープ内にローカル変数 `f` が存在する。
- **期待される出力**: `f "fmt"` への変更は行わず、入力と同じ内容のままにする。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## collision_decl_after_use

- **目的**: 同一ブロック内に同名ローカル変数があっても、その宣言より前の import qualifier 使用だけを rename する場合は安全に auto-fix できる、という DEC-11.22 の精密化候補を確認する。
- **入力の要点**: `fmt.Println` の後にローカル変数 `f` が宣言される。
- **期待される出力**: import が `f "fmt"` になり、宣言前の `fmt.Println` だけが `f.Println` に書き換わる。
- **現状**: `shape.NameVisibleAt` が同一ブロック内の宣言前後を区別しないため、現在は red になる想定。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2、DEC-11.22。

## collision_universe_len

- **目的**: import qualifier を predeclared identifier（例: `len`）へ rename する場合は universe scope との衝突として auto-fix をスキップする、という DEC-11.22 の精密化候補を確認する。
- **入力の要点**: `fmt` が無 alias で import され、`len` へ rename しようとする。
- **期待される出力**: `len "fmt"` への変更は行わず、入力と同じ内容のままにする。
- **現状**: `shape.NameVisibleAt` が universe scope を見ないため、現在は red になる想定。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2、DEC-11.22。

## collision_alias_to_unaliased

- **目的**: alias 付き import を無 alias import へ戻すとローカル識別子と衝突する場合に、危険な auto-fix をスキップできることを確認する。
- **入力の要点**: `f "fmt"` が import され、同じ関数スコープ内にローカル変数 `fmt` が存在する。
- **期待される出力**: 無 alias import への変更は行わず、入力と同じ内容のままにする。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## no_collision_unrelated_scope

- **目的**: 変更対象の参照スコープとは無関係な別スコープの同名識別子を衝突と誤判定せず、auto-fix できることを確認する。
- **入力の要点**: `fmt` が無 alias で import され、別関数 `unrelated` 内にローカル変数 `f` が存在するが、`main` 内の `fmt.Println` とはスコープが重ならない。
- **期待される出力**: import が `f "fmt"` になり、`main` 内の参照が `f.Println` に書き換わる。`unrelated` 内のローカル変数 `f` は変わらない。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## unalias_no_collision

- **目的**: alias 付き import を無 alias import へ戻しても衝突しない場合に、import と参照を安全に unalias できることを確認する。
- **入力の要点**: `f "fmt"` が import され、`f.Println` が使われている。`fmt` と衝突するローカル識別子は存在しない。
- **期待される出力**: import が無 alias の `fmt` になり、参照が `fmt.Println` に書き換わる。
- **根拠 FR・DEC**: FR-6.10、DEC-3.1、DEC-7.2。

## duplicate_import_merge

- **目的**: 単一ファイル内で同一 import path が複数 alias で import されている場合に、package-wide majority で定まった canonical alias へ統合し、余剰 import を削除できることを確認する。
- **入力の要点**: `foo.go` で `fmt` が `f` と `oldfmt` の 2 alias で import され、別ファイル `bar.go` も `f "fmt"` を使うため `f` が canonical になる。
- **期待される出力**: `oldfmt.Println` が `f.Println` に書き換わり、`oldfmt "fmt"` の import が削除される。`bar.go` は変わらない。
- **根拠 FR・DEC**: FR-6.16、FR-7.16、DEC-11.20。

## duplicate_import_collision

- **目的**: 重複 import を canonical alias へ統合するとローカル識別子と衝突する場合に、危険な auto-fix をスキップできることを確認する。
- **入力の要点**: `main.go` で `fmt` が `f` と `oldfmt` の 2 alias で import され、`oldfmt` の利用箇所ではローカル変数 `f` が見えている。別ファイル `other.go` も `f "fmt"` を使うため `f` が canonical になる。
- **期待される出力**: `oldfmt.Println` から `f.Println` への変更は行わず、入力と同じ内容のままにする。
- **根拠 FR・DEC**: FR-6.16、FR-7.16、DEC-11.20、FR-7.5。

## alias_collision_unalias

- **目的**: 同一 alias が複数 import path に対応する場合に、決定ロジックが返す path 順の先頭をその alias のまま残し、残りの path を無 alias import へ振り替えて衝突を解消できることを確認する。
- **入力の要点**: `bar.go` は `x "flag"`、`foo.go` は `x "fmt"` を使っており、package-wide では alias `x` が複数 path に対応している。
- **期待される出力**: path 順で先頭の `flag` は `x "flag"` のまま残し、`fmt` 側は無 alias import に変更して `x.Println` を `fmt.Println` に書き換える。
- **根拠 FR・DEC**: FR-6.11、FR-7.8、DEC-3.1、DEC-11.6、FR-7.5。

## alias_collision_skip

- **目的**: FR-6.11 の衝突解消で無 alias 化後の識別子がローカル識別子と衝突する場合に、危険な auto-fix をスキップできることを確認する。
- **入力の要点**: `bar.go` は `x "flag"`、`foo.go` は `x "fmt"` を使う。`foo.go` の `x.Println` と同じスコープにローカル変数 `fmt` が存在する。
- **期待される出力**: `x.Println` を `fmt.Println` に変更するとローカル変数 `fmt` と衝突するため、入力と同じ内容のままにする。
- **根拠 FR・DEC**: FR-6.11、FR-7.8、DEC-3.1、DEC-11.6、FR-7.5。
