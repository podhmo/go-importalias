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

---

## 第7回（マルチファイル多数決 + 使用箇所診断の実験）で見つかった論点

`docs/02notice.md`第7回：fixのgolden testを本物のマルチファイル入力（scan→decide→fix経由）に拡張し、go vetの診断位置を「使用箇所（fix対象そのもの）」にする実験を行った際に見つかった論点。

### PRE-21: `internal/scan`も型情報（`*types.Info`）を明示引数として受け取る設計を、DEC-11.3の原則として`internal/fix`だけでなく`internal/scan`にも明文で拡張すべきか

- **背景**: DEC-11.3は「`internal/fix`は`*types.Info`/`*types.Package`を呼び出し側からの明示引数として受け取り、`internal/fix`自身は`golang.org/x/tools/go/packages`を呼ばない」と定めていたが、`internal/scan`については型情報の要否に触れていなかった。第7回で、診断を使用箇所単位に出すため`internal/scan.FromFiles`にも`typesInfo *types.Info`引数（nil許容）を追加する必要が生じた。go vet Analyzer（D1）は`pass.TypesInfo`をdriverから無償で受け取れるため影響がなかったが、CLI（D2）側は今後「`internal/fix`向けだけでなく`internal/scan`向けにも型情報付きで`go/packages`をロードする」実装が必要になる。
- **論点**: これは`internal/scan`のAPI契約（型情報を必須にするか、任意にするか）およびCLI（D2）の実装順序（型情報ロードのタイミング）に関わる、パッケージ間の責務分担の話であり、原則レベルの論点と判断した。
- **推奨（デフォルト）**: DEC-11.3の適用範囲を`internal/scan`にも明示的に拡張し、「`internal/scan`・`internal/fix`はともに`*types.Info`を呼び出し側からの明示・任意引数として受け取り、`go/packages`のロードはCLI（`cmd/goimportalias`）の責務に一本化する」と補足する。型情報がnilの場合、`internal/scan`は`Occurrence.UsePos`を空のまま返し（診断はimport宣言行にフォールバック）、go vet Analyzerのように型情報が常に手に入る文脈でも、CLIのように明示ロードが要る文脈でも同じAPIで動作する。

---

## 第8回（import別名とローカル変数名の衝突検出）で見つかった論点

`docs/02notice.md`第8回：`internal/fix`のリネーム処理にFR-7.21相当の識別子衝突チェックを実装し、`fmt→f`・`f→fmt`両方向で衝突時にfixがスキップされることを検証した際に見つかった、衝突判定の精度・範囲に関する論点。

### PRE-22: 識別子衝突チェック（`shape.NameVisibleAt`）の判定範囲をどこまで厳密/保守的にするか

- **背景**: 実装した衝突チェックは2点、意図的に保守的な近似になっている。(a) 同一ブロック内であれば、対象のimport使用箇所がローカル変数の宣言より**前**にあり技術的には安全な場合でも「同名のローカル変数が同じブロックに存在する」というだけで衝突と判定する（Goの「宣言以降スコープに入る」という前後関係を区別しない）。(b) universe scope（`len`・`cap`等の組み込み識別子）との衝突は範囲外としている。
- **論点**: (a)は「疑わしきはfixしない」というtrivial transformationの原則には合致するが、本来fixできたはずのケースを取りこぼす（過剰に保守的な）可能性がある。(b)は多数決の結果として別名が組み込み識別子と同じ文字列になる、という極端だが理論上あり得るケースを無視している。どちらも「衝突判定の完成度」という、`internal/fix`のAPI契約・振る舞いに関わる原則レベルの論点だが、(a)は要件(FR-7.5「安全でなければ直さない」)に照らせば現状の保守的な挙動で十分という考え方もでき、単なる実装の詰めの甘さではなく積極的な設計判断でもある。
- **推奨（デフォルト）**: 現状の保守的な近似（ブロック単位・universe scope対象外）を正式な仕様として採用する。「宣言前後」まで区別する精密な判定や、universe scopeとの衝突チェックは、実際にそれが原因でfixが不必要にスキップされる事例が`docs/fix-cases.md`のケース収集の中で出てきた場合にのみ、対応を検討する。
