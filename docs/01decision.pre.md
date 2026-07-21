# 未確定事項（Pre-Decisions）— 質問リスト

- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/01decision.md`（確定済みの実装決定）を補うための、未確定事項の一時的な置き場。
- **各質問の「推奨（デフォルト）」は実装者側からの提案であり、ユーザーの確認が済むまでは確定事項（DEC-）にはしない。**
- **運用**: 回答・確認が済んだ項目は本書から削除し、`docs/01decision.md`に`DEC-`として追記する。

---

> **確定済みで本書から削除した項目（第5回・PRE-15〜17）**: `docs/draft.md`の`decideOne`（決定ロジック本体）を実際に書き下す中で見つかった3件は、いずれもユーザー確認を経て`docs/01decision.md`のDEC-11.15〜DEC-11.17として確定した。
>
> - PRE-15（設定ファイルのスコープ優先順位判定・`"foo/..."`パターンマッチの置き場所）→ DEC-11.15: `internal/shape`に`Lookup`メソッドを持たせ、複数`"foo/..."`マッチ時は最長prefixを優先（いずれも推奨デフォルト通り）。
> - PRE-16（`AliasCollision`の診断位置情報）→ DEC-11.16: `Occurrences []Occurrence`を持たせる（推奨デフォルト通り）。
> - PRE-17（多数決タイが3つ以上のaliasで発生した場合の表現）→ DEC-11.17: **推奨デフォルト（先頭2件のみ記録）ではなく**、ユーザー判断により型を可変長（`[]string`、2要素以上）に変更し、`DEC-2.1`のスキーマもあわせて更新した。

> **確定済みで本書から削除した項目（第6〜9回・PRE-18〜22）**: `go.mod`を実際に作成し、scan用（analysistest）・fix用（golden file）のテストハーネスを実装・実験する中で見つかった5件は、いずれもユーザー確認を経て`docs/01decision.md`のDEC-11.18〜DEC-11.22として確定した。
>
> - PRE-18（`internal/scan`は`inspector.Inspector`を使うべきか）→ DEC-11.18: 推奨デフォルト通り、`file.Imports`の直接走査を維持し`Requires`は空のまま。
> - PRE-19（blank import・dot importの扱い）→ DEC-11.19: 推奨デフォルト通り、`internal/scan`の時点で多数決・一貫性チェックの対象から除外する（コード側も本ラウンドで対応済み）。
> - PRE-20（`astutil`と直接AST書き換えの使い分け）→ DEC-11.20: 推奨デフォルト通り、同一import path内の別名変更（FR-7.21の別名除去を含む）は常にdirect mutation、astutilはFR-6.11解決（import path自体の追加・削除）だけに使う。第8・9回の実装・golden testで裏付け済み。
> - PRE-21（`internal/scan`にも型情報を明示引数として渡す設計をDEC-11.3として拡張すべきか）→ DEC-11.21: 推奨デフォルト通り。第7回時点で実装・テスト済みだった内容をそのままDEC化。
> - PRE-22（識別子衝突チェックの判定精度）→ DEC-11.22: **推奨デフォルト（現状の保守的な近似を正式仕様として恒久採用）ではなく**、ユーザー判断により「より精密な判定（宣言前後の位置関係・universe scopeとの衝突）へ将来移行する」方針を採用。ただし本ラウンドでは方針の記録のみとし、具体的な実装は別イテレーションに持ち越した（現行の`shape.NameVisibleAt`はそのまま）。

## PRE-23: DEC-11.22 識別子衝突判定の精密化で扱う範囲

**背景**: issue 13 は `shape.NameVisibleAt` の保守的近似を精密化する候補として、変数名・シンボル名の変更が import qualifier rename と衝突するケースを洗い出すもの。現時点では赤テストを増やすのではなく、Go として valid / invalid か、auto-fix で対応すべきか、対応するには何が必要かを整理する。

**実験で確認した Go のスコープ事実**:

- import name は file block に入る。別 import と同じ import name になる rewrite は invalid。
- package-level の `var` / `const` / `type` / `func` と、いずれかのファイルの import name が同名になる package は invalid。これは同一ファイルだけでなく別ファイルでも invalid。
- 関数内ローカル宣言は宣言位置以降だけ有効。したがって同一ブロックでも「import 使用 → 後続で同名ローカル変数宣言」は valid だが、「同名ローカル変数宣言 → import 使用」は invalid / 意味破壊になる。
- inner block・closure・`init` は特別扱い不要で、通常の lexical scope と宣言位置で判定できる。通常関数・メソッド・closure の parameter、named result、type parameter、method receiver は、その関数 body 内で import name を隠す。
- 外側ブロックの同名ローカル変数が closure literal より前で宣言されていれば、closure 内の import 使用も shadow される。closure literal より後の宣言なら、その closure 内からは見えない。
- `for` / `if` / `switch` の init statement で宣言された名前は、それぞれの body / case 内で import name を隠す。`range` 変数も loop body 内で隠す。
- label、struct field、method name、selector の field/method は通常識別子とは別名前空間なので、import qualifier rename とは衝突しない。ただし method receiver 変数名は通常の parameter と同じく衝突し得る。
- predeclared identifier（`len` など）を import alias にすること自体は、ファイル内に既存の builtin 使用がなければ valid。既存の `len(...)` などがあると、alias 後は package name として解決されるため invalid / 意味破壊になる。

**ケース分類**:

| ケース | Go としての結果 | auto-fix 方針 | 対応に必要な判定 |
|---|---:|---|---|
| rename 後の名前が同一ファイルの別 import name と一致 | invalid | skip | file scope の他 `*types.PkgName` を検出 |
| rename 後の名前が package-level decl と一致（同一/別ファイル） | invalid | skip | package scope の object を検出 |
| 使用箇所より前に同一/外側 block の同名 local がある | invalid / 意味破壊 | skip | 使用位置で実際に見える object を検出 |
| 同一 block に同名 local があるが、その宣言は全使用箇所より後 | valid | rewrite 可 | object の宣言位置と使用位置を比較 |
| inner block 内だけに同名 local があり、import 使用は外側だけ | valid | rewrite 可 | innermost scope からの可視性判定 |
| inner block / closure 内の import 使用が parameter・local・type parameter・receiver に隠される | invalid / 意味破壊 | skip | function literal を含む通常 scope 判定 |
| 通常関数・メソッドの引数名、named return、receiver 名、type parameter が import 使用を隠す | invalid / 意味破壊 | skip | 関数 signature が作る scope を使用位置で判定 |
| `init` 内で同名 local と衝突 | 通常関数と同じ | 通常関数と同じ | 特別扱いせず scope 判定 |
| `for` / `if` / `switch` init 変数、range 変数と body 内使用が衝突 | invalid / 意味破壊 | skip | statement-created scope の可視性判定 |
| label / field / method name と同名 | valid | 無視 | `types.Object` の通常スコープに出ないものは衝突扱いしない |
| rename 後の名前が predeclared identifier で、既存 builtin 使用あり | invalid / 意味破壊 | skip | `types.Universe.Lookup(name)` に解決される `info.Uses` を file 全体で検出 |
| rename 後の名前が predeclared identifier だが、既存 builtin 使用なし | valid | rewrite 可（ただし保守的に skip も選択肢） | builtin 使用がないことを確認 |

**対応可能として扱う候補**:

- **精密な可視性判定**: `types.Scope.LookupParent(name, pos)` 相当の、位置を考慮した lookup を使い、同一 block 内の「宣言後だけ衝突」を区別する。これにより、closure・block・`init`・`for/if/switch/range` も同じ仕組みで扱える。
- **package/file block の invalid 化検出**: import spec 自体を rename した時点で package-level decl や別 import と衝突するケースは、使用箇所に関係なく skip する。
- **universe scope の条件付き判定**: rename 後の名前が predeclared identifier の場合、file 内に既存 builtin 使用があるなら skip、なければ valid とみなす。ただし実装を単純化したい場合は v1.1 では predeclared identifier への rename を一律 skip としても安全。

**対応が難しい、または今回扱わない候補**:

- **部分 rewrite**: 1 つの import に対する使用箇所の一部だけが安全で、一部が衝突する場合に、安全な箇所だけを書き換えて import を分割・追加するような変換は行わない。1 import spec の rename は全使用箇所が安全な場合だけ適用する。
- **ローカル識別子側の rename**: import qualifier を通すために既存のローカル変数、関数、型、別 import などを改名する変換は行わない。
- **invalid 入力の救済**: 既に type-check できない入力を、rename で直す/悪化させないように扱うことは対象外。DEC-11.22 は valid input を invalid output にしないための判定に限定する。
- **semantic import alias の是非判断**: `len "fmt"` のような読みづらいが valid な alias を style として禁止するかは、衝突判定ではなく別の policy/config 論点にする。

**推奨（デフォルト）**: DEC-11.22 の実装では「valid input を invalid output にしない」ことを基準に、位置を考慮した scope lookup・package/file block collision・既存 builtin 使用検出を実装する。部分 rewrite とローカル識別子側 rename は非対応とし、1 import spec の全使用箇所が安全な場合だけ rewrite する。
