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

**背景**: issue 13 は `shape.NameVisibleAt` の保守的近似を精密化する候補として、(1) 同一ブロック内の宣言前後を区別すること、(2) universe scope（`len` など）との衝突を扱うこと、の 2 点を挙げている。本ラウンドでは実装ではなく、対応する rewrite / 対応しない rewrite の境界を red test として固定する。

**対応可能として扱う候補**:

- **宣言より前の使用箇所だけを書き換えるケース**: 同一ブロック内に rename 後の識別子と同名のローカル変数があっても、そのローカル変数の `types.Object.Pos()` より前にある import qualifier 使用は、Go のスコープ上まだそのローカル変数が見えていないため安全に書き換え可能とみなす。
- **既存の安全側 skip は維持するケース**: rename 後の識別子が使用箇所で既に見えている場合（宣言後の同一ブロック、外側スコープ、package scope、別 import など）は、これまで通り auto-fix をスキップする。
- **universe scope との衝突を検出して skip するケース**: rename 後の識別子が `len`・`cap` などの predeclared identifier と一致する場合は、ファイル全体で組み込み識別子を shadow し得るため、現時点では使用有無を問わず auto-fix をスキップする。

**対応が難しい、または今回扱わない候補**:

- **部分 rewrite**: 1 つの import に対する使用箇所の一部だけが安全で、一部が衝突する場合に、安全な箇所だけを書き換えて import を分割・追加するような変換は行わない。1 import spec の rename は全使用箇所が安全な場合だけ適用する。
- **ローカル識別子側の rename**: import qualifier を通すために既存のローカル変数、関数、型、別 import などを改名する変換は行わない。
- **universe shadowing の使用箇所精査**: `len` などへの rename が実際に既存の組み込み関数呼び出しを壊すかどうかをファイル全体で精査して条件付き許可することは、今回は扱わない。安全側に一律 skip する。

**追加した red test**:

- `TestApplyToFile_Collision/unaliased_to_alias_declared_after_use_is_safe`: `testdata/fix/collision_decl_after_use`。期待値は rewrite ありだが、現状の `NameVisibleAt` は同一ブロック内の宣言前後を区別しないため `changed=false` になり red。
- `TestApplyToFile_Collision/unaliased_to_universe_name_is_collision`: `testdata/fix/collision_universe_len`。期待値は rewrite なしだが、現状の `NameVisibleAt` は universe scope を見ないため `changed=true` になり red。

**推奨（デフォルト）**: 上記の「対応可能として扱う候補」を DEC-11.22 の実装範囲とし、「対応が難しい、または今回扱わない候補」は明示的に非対応とする。今回追加したテストは意図的に red のままにし、次イテレーションで `NameVisibleAt` を精密化する際の受け入れ条件として使う。
