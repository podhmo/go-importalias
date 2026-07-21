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

現時点で未解決のPRE項目はない。次のドラフト作業（`docs/draft.md`をさらに掘り下げる、または`internal/scan`・`internal/decide`・`internal/fix`のテストハーネス試作に着手する等）で新たな論点が見つかった際に、本書を再度使う。

---

## 第6回（テストハーネス実装実験）で見つかった論点

`docs/02notice.md`第6回：`internal/shape`/`internal/scan`/`internal/decide`/`internal/fix`を実際にコードとして書き、scan側（analysistest）・fix側（golden file）それぞれ1ケースずつテストをgreenにする実験を行った際に見つかった、原則レベルの未決事項。テストハーネス自体の機械的な仕組み（golden比較の方式、型情報取得方法の使い分け等）はADR-00の基準に従いここには含めず、`docs/02notice.md`第6回の知見側に留めている。

### PRE-18: `internal/scan`は`go/ast/inspector.Inspector`を使うべきか、`ast.File.Imports`の直接走査で十分か

- **背景**: `docs/draft.md`では`inspector.Inspector`（および`Analyzer.Requires: []*analysis.Analyzer{inspect.Analyzer}`）を使う設計だったが、「使う理由が言語化されていない」と保留されていた。第6回の実験では、import宣言だけを対象とする限り`file.Imports`の直接走査で`analysistest`によるFR-6.10検知が成立し、`Requires`を空にできた。
- **論点**: これは単なる実装詳細ではなく、Analyzerの`Requires`（他のAnalyzerとの依存関係・multichecker上での計算共有）という外部契約に関わる。generated file判定など将来AST全体を舐める処理が増えた場合に再度必要になる可能性もある。
- **推奨（デフォルト）**: 現状（import宣言のみを扱う）では直接走査を採用し、`Requires`は空のままとする。他の解析ニーズが具体化した時点で改めて`inspector.Inspector`導入を検討する。

### PRE-19: blank import（`_`）・dot import（`.`）を`internal/scan`/`internal/decide`でどう扱うか

- **背景**: 第6回の実装では、`internal/scan.FromFiles`がblank import・dot importもそのまま`Occurrence.Alias`（`"_"`または`"."`）として収集してしまい、`internal/decide`の多数決にそのまま混入する。`docs/00origin.md`・`docs/01decision.md`のどちらにも、この2種類のimportをどう扱うべきかの記載がない。
- **論点**: これらは通常「別名の揺れ」を議論する対象ではなく、多数決の候補に含めると誤った"correct"判定を生みかねない、`shape.Occurrence`/`decide`の意味論に関わる設計原則レベルの問題。
- **推奨（デフォルト）**: `internal/scan`の時点で`Alias`が`"_"`または`"."`のoccurrenceは多数決・一貫性チェックの対象から除外する（別途保持するか、単純に無視するかは実装時に決める）。

### PRE-20: import path自体の追加・削除を伴うfixケースで、`astutil`と直接AST書き換えをどう使い分けるか

- **背景**: DEC-11.6は`internal/fix`が`astutil.AddNamedImport`/`DeleteNamedImport`を使う設計としていたが、第6回で実装した「同一importパスのまま別名だけを直す」最小ケースでは、既存`*ast.ImportSpec.Name`を直接書き換えるだけで足り、astutilは不要だった。
- **論点**: FR-6.11の解決（別名→複数pathの片方に寄せる）やFR-7.21（別名除去＋識別子衝突チェック）では、import宣言そのものの追加・削除が必要になり、direct mutationでは足りずastutilが要ると予想される。`internal/fix`内で「単純リネームはdirect mutation、path追加/削除はastutil」と実装を使い分けるのか、一貫性のために常にastutil経由にするのか、DEC-11.6の適用範囲を明確にする必要がある。
- **推奨（デフォルト）**: 単純な別名リネーム（importパス不変）はdirect mutationのまま維持し、importパスの追加・削除を伴うケースのみastutilを使う、と使い分けを明文化する（DEC-11.6を「path追加/削除を伴う場合」に限定するかたちで補足する）。
