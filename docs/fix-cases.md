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
