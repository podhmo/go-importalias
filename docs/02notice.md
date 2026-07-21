# 実装ドラフト作業メモ（第1回）

- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/00origin.md`・`docs/01decision.md` を元に、DEC-10.1のステップ1（`internal/config`: DEC-2.1の型定義とMarshal/UnmarshalJSON）を実際にコードとして書いてみた際に思ったこと・気づいたことのログ。ドラフト本体は `docs/draft/config.go`。ここで見つかった論点のうち未確定なものは `docs/01decision.pre.md` に切り出した。

## やったこと

`AliasValue`（`Resolved string` / `Tie []string`）と `File`（`Packages` / `Ignore`）の型、およびカスタム `MarshalJSON`/`UnmarshalJSON`、加えて `Load`/`Save` という設定ファイルI/O関数を1ファイルにまとめて試作した。

## 気づいたこと

1. **DEC-2.1は「型定義のみ」しか明記していない**。`Load`/`Save`のような読み書き関数の存在・シグネチャは書かれていないが、DEC-7.3（round-tripテスト）やFR-7.3（「存在しない場合は新規作成」）を実際に動かすには、どこかにファイルI/Oの入口が必要になる。`internal/config`に置くのが一番自然に思えたので、そう仮定してドラフトに含めた。

2. **`packages`/`ignore`キーのソート（DEC-2.2）は、mapを使っている限り追加コード不要でほぼ自動的に満たされる**。`encoding/json`は`map[string]...`をMarshalする際にキーをアルファベット順にソートする仕様のため。明示的にソートコードを書く必要があったのは`AliasValue.Tie`の2要素配列だけだった。

3. **nilマップの罠**：`config.File{}`をゼロ値のまま直接組み立てると`Packages`が`nil`になり、`encoding/json`は`nil`のmapを`"packages": null`として出力する。これは5.9節の全体像の例（`"packages": {...}`）が期待する見た目と食い違う。`Load()`経由なら空map初期化を仕込めるが、テストコード等で`File{}`を直接組み立てるケースをどう扱うかは決めていなかったので、`NewFile()`ヘルパーを追加した。

4. **エラーメッセージのスタイルが何も決まっていない**ことに気づいた。日本語話者向けのプロジェクトだが、Goの一般的な慣習（`errors`パッケージのガイドライン：小文字始まり、`%w`でラップ）に倣って英語でエラー文字列を書くべきか、日本語で書くべきかは`docs/00origin.md`・`docs/01decision.md`のどちらにも記載がなかった。ドラフトでは一旦Go標準寄り（英語・`importalias: `プレフィックス）を採用した。

5. **`AliasValue`が「あってはならない状態」（`Resolved`と`Tie`が両方非ゼロ値）を型として防げていない**ことに気づいた。今回は`NewFile`と同様に`Resolved(alias)`/`Tie(a, b)`というコンストラクタ関数を用意し、呼び出し側がこの2つ経由でのみ値を作るように誘導する方向で応急処置したが、これがコード全体の方針として十分かは検証していない。

6. **go.mod・ディレクトリ骨格がまだ存在しない**ため、今回のドラフトは`docs/draft/config.go`という「ビルドされない参考コード」の位置づけで留めた。DEC-10.1ステップ1に本格着手する際は、まず`go.mod`の作成（DEC-1.1のモジュールパス・DEC-1.2のGoバージョン）と`internal/config`ディレクトリの作成が前提として必要になる。これは実装着手順（10節）に明示的な項目としては存在しない、ステップ0的な作業だと感じた。

7. **`ignore`パターン（`foo/...`）の判定ロジックをどこに置くか**を書こうとしたところで、`internal/config`の責務（「型・読み書き・マージ」）を超えると判断して止めた。マッチング処理自体は走査（`internal/scan`）側の責務だろうと推測しているが、明文化はされていない。

## 次にやること（第1回時点のメモ・第2回で方針転換）

- `docs/01decision.pre.md`の質問に回答（またはデフォルト採用の確認）が付いたら、`docs/draft/config.go`を`internal/config/config.go`として本実装に格上げし、`go.mod`を作成した上でDEC-7.3のround-tripテストを書く。
- その次のイテレーションでは、`internal/config`のテストを書く過程でさらに論点が出るはずなので、同じサイクル（ドラフト→気づき→pre.md）を継続する。

---

# 第2回：方針転換 — パッケージ詳細ではなく設計原則を洗い出す

- **日付**: 2026-07-21（同日、第1回の直後）
- **ユーザーからのフィードバック**: 第1回でやった「`internal/config`という1パッケージの実装詳細（`Load`の挙動、ファイルパーミッション等）を細かく詰める」というアプローチは求めていたものと違う、という指摘を受けた。決めたいのは**設計原則・コーディング原則**であり、かつ「潜在的にまだ決めていないこと」をあぶり出すのが目的。そのための手段として、1パッケージを丁寧に書くのではなく、`draft.md`という1ファイルに**repomix的**（＝リポジトリ全体を1つのテキストにまとめる形式）に、`go.mod`〜`cmd/`〜`internal/*`をラフに一通りスケッチし、各断片に「これは動くはず（検証済み/未検証）」「ここはこう設計するはず（未検証・要確認）」という確信度の注記を付けていく、というやり方に転換した。

## 第1回との違い（今後もこの粒度感を維持する）

- **第1回**: 1パッケージ（`internal/config`）を実際にビルド・vetが通るところまで作り込み、そこで出た細かい仕様の揺れ（ファイルパーミッション、エラーメッセージの体裁等）を質問にした。→ 粒度が細かすぎた。
- **第2回**: リポジトリ全体を薄くスケッチし、**パッケージ同士の繋がり方・依存の向き・責務の切り方**という、後から変えるとコストが大きい部類の論点だけを拾い上げた。個々の関数のちょっとした挙動（PRE-2, PRE-3等、旧版）は今回は意図的に扱わない。

## draft.mdを書く過程での気づき（原則以外の技術的発見も含む）

1. **`unitchecker.Main`は`os.Args`を自分でグローバルに読みに行く**（ソース: `golang.org/x/tools/go/analysis/unitchecker/unitchecker.go`、`analysisflags.Parse(analyzers, true)` → `flag.Args()`という順）。つまり`cmd/goimportalias/main.go`のモード分岐（vetツールモード or CLIモード）は、`unitchecker.Main`を呼ぶ**前に**、こちらで`os.Args`を覗き見るだけで判定し、`flag.Parse()`などを事前に呼んではいけない（二重パースになる）と確認できた。これはDEC-1.4が「具体的な判定条件は実装時に固める」としていた部分の、実装可能性の裏取りになった。

2. **`analysis.Pass.Module`は実在するが、ドキュメント上「possibly nil in some drivers」と明記されている**（`go doc golang.org/x/tools/go/analysis Pass`で確認）。D1（go vet analyzer）がモジュールルートの`importalias.json`を読もうとする場合、この`Module.Dir`に頼るのは環境依存のリスクがあると分かった。これが「D1は設定ファイルを読まない」という推奨（`01decision.pre.md` PRE-2）の直接の根拠になっている。

3. **`astutil.AddNamedImport`/`DeleteNamedImport`は実在するが、qualified identifier（`pkg.Symbol`）の書き換えを助けるヘルパーはastutilには無い**。FR-7.7・FR-7.21が要求する「識別子の安全な書き換え」は、`go/types`の型情報を`internal/fix`にどう受け渡すかという、パッケージ間のデータフロー設計が抜けていることに気づいた（PRE-3）。

4. **型・関数を実際に書き始めて初めて、パッケージ同士のデータの受け渡し方が決まっていないことに気づく**、というパターンが今回の一番の収穫だった。個々のAPIが動くかどうかより、「`scan`の出力を`decide`がどう受け取るか」「`decide`の出力を`fix`がどう受け取るか」という繋ぎ目の設計が抜けていた（→ PRE-1の`internal/model`新設案）。

5. FR-6.10（同一path→複数alias）とFR-6.11（同一alias→複数path）は、`decide`側で同じ集計データ構造だけでは両方をカバーできない（path方向のグルーピングとalias方向のグルーピング、最低2系統が要る）ことに、`decideOne`を書こうとして気づいた。これは`docs/01decision.md`にも`docs/00origin.md`にも明記されていない実装上の必然で、次のイテレーションで`internal/decide`のドラフトをもう少し深掘りする際に扱う。

## 次にやること（第2回時点のメモ）

- `docs/01decision.pre.md`（第2回・原則レベル）の質問に回答が付いたら、`docs/01decision.md`に`DEC-`として追記し、`docs/draft.md`を実際の`internal/*`ディレクトリ構成に格上げする作業を始める。
- 保留：ユーザーから「`02notice.md`自体の構成も見直した方がよい」という指摘を受けている。今は第1回・第2回と単純に追記が積み上がる構成になっているが、この先イテレーションが増えると読みにくくなるはずなので、後日（今回のドラフト作業がひと段落したタイミングで）見出し構成そのものを見直す。

---

# 第3回：PRE-1〜PRE-10確定を`docs/draft.md`に反映し、もう一段深掘り

- **日付**: 2026-07-21（同日）
- **やったこと**: `docs/01decision.pre.md`のPRE-1〜PRE-10すべてがユーザー確認によりDEC-11.1〜DEC-11.10として確定したのを受け、`docs/draft.md`を書き直した。単なる用語置換（`internal/model`→`internal/shape`、`internal/config`廃止）だけでなく、決定した原則を実際にコードへ反映しようとして、以下の一段深い箇所で新たな未決事項が見つかった。

## 気づいたこと

1. **`internal/decide`のFR-6.10/FR-6.11集計を実際に2系統（`byPath`/`byAlias`）に分けて書いてみると、`byAlias`側の検出結果を`shape.Decision`型にどう反映するかが決まっていない**ことが分かった。第2回時点では「2系統の集計が要る」ことまでしか分かっていなかったが、実際に書くと「型として何を返すか」という次の層の問題が出てきた。
2. **`shape.Merge`（DEC-11.8の実装）を実際に書くと、「スキャン範囲内のpackageキーの中でのpath単位の粒度」が未決だと分かった**。DEC-11.8はpackageキー単位の話までしか決めておらず、同一package内で「今回freshに現れなかったimport path」の既存tie指定を消してよいかは別問題として残っている。
3. **`internal/fix`のSelectorExpr走査を`types.Info.Uses`で安全確認する設計（DEC-11.6）自体は実装できたが、「ファイル全体を毎回舐める」実装は非効率・誤爆リスクがある**ことが、実際にコードを書いて初めて分かった。対象importに紐づくSelectorExprだけを事前に絞り込む設計がまだ無い。
4. **`Occurrence.Alias`の「無alias」表現（空文字か、宣言パッケージ名と一致する文字列か）は、第1回のドラフトから一貫して未決のまま持ち越されている**ことに気づいた。地味だが`scan`/`decide`両方が前提にする規約なので、早めに確定させた方がよさそうな候補。

## 次にやること

- 上記4点を新たな`docs/01decision.pre.md`（PRE-11〜PRE-14）として整理するか、あるいはDEC-7.5と同じ「実装しながら都度確定させる」対象にするかを、ユーザーに確認する。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し）。

---

# 第4回：PRE-11〜PRE-14確定後のビルド検証

- **日付**: 2026-07-21（同日）
- **やったこと**: PRE-11〜PRE-14がDEC-11.11〜DEC-11.14として確定し`docs/draft.md`に反映された後、その反映が本当にコンパイル可能な形になっているかを裏取りするため、`docs/draft/`という一時ディレクトリに`docs/draft.md`中の全コードブロック（`go.mod`・`analyzer.go`・`cmd/goimportalias/main.go`・`internal/shape/model.go`・`internal/shape/config.go`・`internal/scan/scan.go`・`internal/decide/decide.go`・`internal/fix/fix.go`・`internal/genfile/genfile.go`）をそのまま書き出し、`go mod tidy`で依存（`golang.org/x/tools v0.48.0`とその推移依存）を解決した上で`go build ./...`・`go vet ./...`を実行した。

## 気づいたこと

1. **一度も実際にコンパイルを試していなかった`internal/fix/fix.go`（DEC-11.3・DEC-11.6・DEC-11.14反映後の`go/types`ベースの書き換えロジック・事前インデックス化）を含め、9ファイルすべてが`go build ./...`・`go vet ./...`ともにエラーなしで通った**。`decideOne`・`lookupPkgNameForPath`・`runCLI`の3関数は中身が`panic("not implemented in this sketch")`のスタブのままだが、Go言語的にはこれは有効な関数本体であり、コンパイルを妨げない。つまり今回確認できたのは「型・シグネチャ・パッケージ間の依存関係（import）が矛盾なく繋がっている」ことであり、実行時の振る舞い（実際に正しい診断が出る、正しく書き換わる等）はまだ何も検証していない。
2. **エディタのgopls（バックグラウンドlint）が`internal/shape/config.go`の`Merge`関数について「2つの`for range`ループを`maps.Copy`に置き換えられる」と提案してきた**が、これは`go vet ./...`自体の指摘ではなくgoplsの追加的な静的解析（スタイル上の指摘）であり、区別して記録する必要があると気づいた。`go vet`単体を叩いた際の標準出力は空（エラーなし）だった。
3. 検証後、一時ディレクトリ（`docs/draft/`、`go.mod`・`go.sum`含む）は成果物として残さず削除した。`docs/draft.md`という単一ファイルが実際の成果物である、という第2回で確立した方針を維持している。

## 次にやること

- 上記のビルド検証で見つかった通り、`lookupPkgNameForPath`の未実装、および`analyzer.go`の`AliasCollision`診断（`Pos`をどう選ぶか）の2点は、原則レベルの論点ではなく実装着手時にテストハーネスを整えながら詰める対象と判断し、新たな`PRE-`は起票しなかった（`docs/draft.md`の「現時点での残課題」節に明記済み）。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第5回：残っていたスタブ（`lookupPkgNameForPath`・`AliasCollision`診断・`decideOne`）の実装

- **日付**: 2026-07-21（同日）
- **やったこと**: 第4回で「原則レベルではない」と判断し据え置いていた`lookupPkgNameForPath`と`analyzer.go`のAliasCollision診断に加え、`decideOne`（4節の決定ロジック本体）も実際に書き下した。

## 気づいたこと

1. **`lookupPkgNameForPath`は実装の余地がほぼなく、質問化する必要はなかった**。`go doc go/types Info`で`Implicits`（"`*ast.ImportSpec` → 無aliasの`*PkgName`"）・`Defs`（明示alias付きimportの名前識別子が定義される場所）を確認したところ、importのpathからその束縛先を逆引きする方法は1通りしかなく、単に実装すればよいだけだった。同様に`scan.Occurrence.Package`の穴埋めも、`Options.Package`を足して呼び出し側（`analyzer.go`）が`pass.Pkg.Path()`を渡す以外の妥当な設計が見当たらず、質問化しなかった。これは「実装しながら初めて分かるが、選択の余地自体はない」という、第1回で扱っていた粒度（1関数の実装詳細）に近いケースであり、原則レベルの論点とは区別する必要があると再確認した。
2. **`decideOne`（多数決＋config優先順位＋タイ処理）を実際に書こうとして、`decide`パッケージが「configの優先順位判定ロジックをどこまで知るべきか」を全く決めていなかったことに気づいた**。第2回のdecide.goに残されていたコメント（「shape側に`Lookup`メソッドを生やすのか」という疑問）は、第2回で3点にまとめた際にPRE-1〜PRE-10のいずれにも含まれておらず、第3回・第4回でも見過ごされたまま残っていた。実際に`decideOne`の中身を書こうとすると、この疑問を素通りできないことが分かった。
3. **`AliasCollision`（FR-6.11の検出結果）が位置情報を一切持たない型のままだったため、`analyzer.go`側で`pass.Reportf`を呼びようがない**ことに気づいた。第3回時点の型（`Paths []string`）は「どのpathが衝突しているか」までしか表現できておらず、「診断としてどこに報告するか」という要件を満たせていなかった。
4. **多数決タイが3つ以上のaliasで発生するケースを、`shape.Decision.TieCandidate [2]string`という固定長の型では表現できない**ことに気づいた。`docs/00origin.md` FR-4.6は「最多の組が複数存在する場合は自動確定しない」としか言っておらず、3つ以上のケースを想定していない。実務上は稀なケースだと思われるが、型として表現できない以上、少なくとも「どう振る舞うべきか」を決めておく必要がある。
5. 上記2〜4を`docs/01decision.pre.md`にPRE-15〜PRE-17として整理し、ユーザーに確認、`docs/01decision.md`のDEC-11.15〜DEC-11.17として確定した（詳細はそちらを参照）。反映後、本ドラフト全体（9ファイル）を一時ディレクトリに書き出し、`go mod tidy`→`go build ./...`→`go vet ./...`がすべてエラーなく通ることを再確認した。

## 次にやること

- `internal/decide`・`internal/fix`はコンパイルは通るが、テストを一切書いていないため実行時の振る舞いは未検証。DEC-10.1のステップ順（テストハーネス確立が最優先）に従い、次はドラフトをさらに深掘りするより先に、テストハーネスの試作に進むべきタイミングに来ている可能性がある。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第6回：scan/fixテストハーネスを実際に実装し、`go test ./...`をgreenにする実験

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: これまでの5回はすべて`docs/draft.md`（ビルド確認のみ・実行されない参考スケッチ）または`docs/draft/`という一時ディレクトリ止まりだった。今回は初めて`go.mod`を作成し、生きたモジュール直下に`internal/shape`・`internal/scan`・`internal/decide`・`internal/fix`・ルート`analyzer.go`を実装し、DEC-10.1のステップ1〜3（＋最小限のステップ4・7）に相当する「scanのテストハーネス（analysistest）」と「fixのテストハーネス（golden file）」を実際に動かして`go test ./...`をgreenにする実験を行った。スコープは意図的に絞り、FR-6.10（同一path→複数alias）1ケースの検知と、その1ケースの自動修正のみを対象とした。FR-6.11・FR-6.16・generated file skip・CLI本体は今回対象外。

## やったこと

1. `go.mod`（`module github.com/podhmo/go-importalias`, `go 1.26`, `golang.org/x/tools v0.48.0`）を作成。
2. `internal/shape/model.go`（`Occurrence`/`Decision`/`AliasCollision`）と`internal/shape/config.go`（`File`/`AliasValue`のカスタムJSON、`NewFile`/`Load`/`Save`/`Lookup`/`Merge`）を実装し、round-tripテスト・`Lookup`優先順位テスト・`Merge`テストを`internal/shape/config_test.go`に書いた。
3. `internal/scan/scan.go`（`FromFiles`）・`internal/decide/decide.go`（`Decide`/`decideOne`、FR-6.10のみ）・ルート`analyzer.go`（`var Analyzer`、`Requires`なし）を実装し、`testdata/src/dup/{a,b,c}.go`と`analyzer_test.go`（`analysistest.Run`）で1ケースを検知させた。
4. `internal/fix/fix.go`（`ApplyToFile`、既存importの別名を多数決結果にリネームするだけの最小実装）を実装し、`testdata/fix/rename_to_majority/{input,golden}/main.go`と`internal/fix/fix_test.go`（`go/types`単体チェック＋golden比較）で1ケースをgreenにした。
5. `go build ./...`・`go vet ./...`・`go test ./...`が最終的にすべてエラーなし・全テストPASSであることを確認した。

## 気づいたこと

1. **`inspector.Inspector`は今回のスコープでは本当に不要だった**。`internal/scan.FromFiles`は`file.Imports`を直接ループするだけで実装でき、ルート`analyzer.go`の`Analyzer.Requires`も空（`inspect.Analyzer`への依存なし）のまま`analysistest.Run`によるFR-6.10検知が成立した。`docs/draft.md`で「使う理由が言語化されていない」と保留されていた論点は、少なくとも「import宣言だけを見る」範囲では不要という形で経験的に解消した。ただし将来generated file判定（コメント走査）等でAST全体を舐める処理が増えた場合に再浮上する可能性はある（→PRE-18）。
2. **`astutil.AddNamedImport`/`DeleteNamedImport`を使わず、既存の`*ast.ImportSpec.Name`を直接書き換えるだけで「別名のリネーム」は成立した**。DEC-11.6はastutilの利用を前提にしていたが、今回試した「同一importパスのまま別名だけを直す」という限定ケースでは、import宣言の追加・削除に相当する複雑さは不要だった。ただしこれは限定ケースでの簡略化であり、FR-6.11の解決やFR-7.21（別名除去＋衝突チェック）のようにimport path自体の追加/削除が要るケースでは、direct mutationでは足りずastutilが必要になると予想される（→PRE-20）。また、import行に行末コメントが付いている場合の位置情報保持は今回のfixtureにコメントを置かなかったため未検証のまま。
3. **fixのgolden testにおける型情報取得は、`go/packages`を使わず`go/parser.ParseFile` + `go/types.Config{Importer: importer.Default()}.Check(...)`という単体ファイルの型チェックだけで足りた**。標準ライブラリのみをimportする最小ケースに限れば、`go/packages`のモジュール解決・キャッシュといった重さを回避できることが実験で確認できた。一方で、複数ファイル間でのシンボル参照や外部モジュールの型情報が必要になる本格的なfixケースでは、単体ファイルチェックでは他ファイルの型情報が見えないため成立せず、`go/packages`（または対象パッケージの全ファイルをまとめて`types.Config.Check`に渡す方式）へ切り替える必要が出てくると見込まれる。「単体ファイルチェックで足りるケース」と「複数ファイルロードが要るケース」の境界線は、`docs/fix-cases.md`を書き始める際にケースごとに注記すべき運用上の論点であり、DEC-7.5が言う「実装しながら決める」対象そのものと判断し、新たなPRE化は見送った。
4. **fixのgolden test実行方式は、`cmd/goimportalias`バイナリを実際に起動する（`os/exec`等）のではなく、`internal/fix.ApplyToFile`をin-processで直接呼び出し、返り値の`[]byte`をgoldenファイルとバイト比較するだけで成立した**。DEC-10.1ステップ3が想定していた「minimal cmd/goimportalias CLI」を経由しない設計。CLIレイヤー（モジュールルート探索・config読み書き・複数パッケージの逐次処理）は、この粒度のfixロジック単体テストの対象外にできることが分かった。これもDEC-7.5の「実装しながら決める」対象と判断し、PRE化は見送った。
5. **`internal/scan.FromFiles`は、blank import（`_`）やdot import（`.`）もそのまま`Occurrence.Alias`に入れてしまう**ことに気づいた（今回のfixtureでは意図的に避けたため実害はなかったが、実装上は無防備）。これらは多数決の候補に含めるべきではない特殊ケースだが、`docs/00origin.md`・`docs/01decision.md`のどちらにも明記がなく、今回も対応していない（→PRE-19）。
6. **`analysistest`の`// want`コメントは、診断が報告された行と同じ行に置く必要がある**ことを実地で確認した。`Occurrence.Pos`は`imp.Pos()`（`*ast.ImportSpec`の位置：明示aliasがあればそのalias名の位置、なければpath文字列の位置）から取っているため、`// want`はimport宣言の行に置く必要があり、実際にその識別子を使っている行（`fmt.Println(...)`等）には置けない。最初に使用箇所へ`// want`を置いて1回テストが赤くなり、import宣言の行に移して green になった。
7. **goplsが`internal/shape/config.go`の`Merge`実装（2つの`for range`ループ）に対し、`maps.Copy`への置換を再度提案してきた**。これは第4回でドラフト段階で観測した指摘と全く同じもので、今回は実際に`maps.Copy`へ書き換えた上で`go test`がgreenのままであることを確認した（再現性のある、素直に採用してよい指摘だと判断）。
8. 上記のうち、「原則レベル（他パッケージとの責務分担・データ型設計に関わり、後から変えるとコストが大きい）」に該当する論点（PRE-18〜PRE-20）のみを`docs/01decision.pre.md`に切り出した。「実装しながらでないと判断できないテストハーネスの機械的な仕組み」（項目3・4）はADR-00の基準に従いPRE化せず、本ノートの知見に留めている。

## 次にやること

- `docs/01decision.pre.md`のPRE-18〜PRE-20についてユーザー確認を得て、`docs/01decision.md`にDEC-として追記する。
- FR-6.11（同一alias→複数path）・FR-6.16（同一ファイル内重複import）・generated file skip・CLI本体（`cmd/goimportalias`）は今回スコープ外のまま。次のイテレーションで対象を広げる際は、今回確立したscan/fixそれぞれのテストハーネス（`testdata/src/<pkg>`+`analysistest`、`testdata/fix/<case>`+golden比較）にケースを追加していく形で進められる見込み。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第7回：多数決をマルチファイルで検証 + 診断位置を「使用箇所」にする実験

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: 第6回への2点のフィードバックを受けた追加実験。(1) 多数決の正しさを確認するにはfix側もscan/decideを経由した本物のマルチファイル入力でテストすべき（第6回のfixテストは`shape.Decision`を手で組み立てた1ファイルのみのケースだった）。(2) go vetの診断表示位置を、import宣言行だけでなく「実際にその別名が使われている箇所（＝fixが書き換える箇所）」にすべき（例: foo.go/bar.goで`x`、boo.goで`oldx`と別名を付けている場合、`oldx`の使用箇所自体を診断位置として出したい）。

## やったこと

1. `shape.Occurrence`に`UsePos []token.Pos`を追加。ある importが実際に使われている箇所（`pkg.Symbol`形式の参照）の位置一覧を保持できるようにした。
2. `internal/shape`に`PkgNameOf`（`*ast.ImportSpec`から`*types.PkgName`を解決）・`SelectorIdentsOf`（`*types.PkgName`を参照している`*ast.SelectorExpr.X`識別子を列挙）という2つの小さなヘルパーを新設した。これは元々`internal/fix`だけに書いていたロジックだが、今回`internal/scan`も同じ解決が必要になったため、重複を避けて`internal/shape`に寄せた。
3. `internal/scan.FromFiles`に`typesInfo *types.Info`引数を追加（nil許容）。型情報が渡された場合のみ、各`Occurrence`の`UsePos`を埋める。
4. ルート`analyzer.go`を、`pass.TypesInfo`（Analyzerの実行時にdriverがすでに計算済みで無料で使える）を`scan.FromFiles`にそのまま渡すように変更し、診断を「`Inconsistent`な occurrence の`UsePos`一つひとつ」に対して`pass.Reportf`するように変更した（`UsePos`が空の場合のみimport宣言行にフォールバック）。
5. `internal/fix/fix.go`の`renameImportSpec`を、重複していたPkgName解決・SelectorExpr走査ロジックを`shape.PkgNameOf`/`shape.SelectorIdentsOf`呼び出しに置き換えてリファクタリングした（振る舞いは変えていない）。
6. `testdata/src/dup/b.go`の`// want`コメントを、import宣言行から実際の使用箇所（`fmt.Println(...)`の各行）に移し、かつ使用箇所を2つに増やして「1ファイル内の複数の使用箇所それぞれに診断が出る」ことを検証した。
7. ユーザーの例（foo.go/bar.goで`f`、boo.goで`oldf`）を模した`testdata/fix/multi_file_majority/{input,golden}/{foo,bar,boo}.go`を新設し、`internal/fix/pipeline_test.go`で`scan.FromFiles`→`decide.Decide`→`fix.ApplyToFile`を3ファイル分まとめて型チェックした上で実行し、(a) 多数決が正しく`f`（2票）に決まること、(b) 負けた`boo.go`の`Inconsistent`occurrenceの`UsePos`が2件（`oldf.Println`の2箇所）であること、(c) `foo.go`/`bar.go`は`changed=false`で無変更、`boo.go`だけが`changed=true`でgoldenと一致することを確認した。

## 気づいたこと

1. **`go vet`のAnalyzer（D1）は`pass.TypesInfo`をdriverからタダで受け取れる**ため、型情報を使った使用箇所解決（`PkgNameOf`/`SelectorIdentsOf`）にAnalyzer側で`go/packages`を呼ぶ必要は一切なかった。一方CLI（D2）側では、DEC-4.3がすでに「`internal/fix`向けに`go/packages`で型情報をロードする」ことを決めていたが、今回`internal/scan`も型情報を使う設計に変わったため、「`internal/scan`にも同じ型情報を渡す」という一手間がCLI側の実装に増えることが分かった（→PRE-21）。
2. **`internal/fix`にだけ書いていた「PkgNameを解決してSelectorExprを走査する」ロジックが、`internal/scan`にも実質同じ形で必要になり、コードの重複が生まれかけた**。今回は`internal/shape`（すでにscan/decide/fix共通のドメイン型置き場と位置づけられていた、DEC-11.1）に`PkgNameOf`/`SelectorIdentsOf`という薄いヘルパーとして寄せることで解消した。DEC-11.1の「shapeは共有ドメイン型の置き場」という原則が、データ型だけでなく「その型が持つ意味を扱う小さな共通処理」にも自然に拡張できることが実地で確認できた。
3. **診断位置を使用箇所単位にしたことで、`analysistest`の`// want`コメントも使用箇所単位（1ファイル内で複数行）に分割する必要があった**。第6回で確認した「`// want`は診断と同じ行に置く」というルールの延長で、1つのoccurrenceが複数の使用箇所を持つ場合は、その数だけ`// want`を用意する必要がある、という運用が確認できた。
4. **マルチファイルの型チェックは、`types.Config.Check(pkg, fset, []*ast.File{...全ファイル...}, info)`に対象パッケージの全ファイルをまとめて渡すだけで、`go/packages`なしで成立した**（第6回の単体ファイルチェックの延長）。今回は標準ライブラリ（`fmt`）のみに依存するため成立しており、外部モジュールへの依存や複数パッケージにまたがるケースでは引き続き`go/packages`が必要になる境界は変わっていない。
5. `fix.ApplyToFile`は`Occurrence.UsePos`を直接使わず、`shape.PkgNameOf`+`shape.SelectorIdentsOf`をその場で呼び直して書き換え対象を求めている。診断表示用（scan/analyzer側）と書き換え用（fix側）で「使用箇所を求める」という同じ計算を2回行っている形になっており、`UsePos`をfix側に渡して再利用する設計に寄せるべきかは未検討のまま残っている。

## 次にやること

- 上記1点（PRE-21）を`docs/01decision.pre.md`に追記した。
- 気づき5（診断用と書き換え用で使用箇所解決を2回行っている点）は、原則レベルというよりは効率上の最適化余地に近いと判断し、今回はPRE化を見送った。将来的にファイルサイズが大きくなった場合の性能上の懸念として、DEC-11.14（`internal/fix`の事前インデックス化方針）と合わせて扱うのがよさそうという所感のみ残す。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第8回：qualified importの別名とローカル変数名の衝突を実装・検証する実験

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: ユーザーから追加で「importの別名と関数内変数名が衝突するケースをきちんと考えて欲しい」という指摘を受けた実験。指摘は2点に整理できる：(1) 既存コードで変数名がimportの別名と衝突（同名でシャドーイング）していても、それ自体は普通に動くGoコードであり、我々のツールのfix対象にはならない（＝既存の`shape.SelectorIdentsOf`の`types.Info.Uses`ベースの解決が、シャドーされた位置を正しく除外できているはず）。(2) しかし fix によって**新しく導入される別名**が、たまたま既存の変数名と衝突するケースは要注意で、`fmt`（無alias）→`f`の方向と、`f`→`fmt`（無alias化）の方向の両方を確認する必要がある。

## やったこと

1. `internal/fix.ApplyToFile`の`WantAlias == ""`（無alias化）の扱いに**バグがあった**ことに気づき修正した。旧実装は`renameImportSpec`で単純に`ident.Name = newAlias`としており、`newAlias == ""`のとき全ての使用箇所識別子の名前を空文字列にしてしまっていた（`fmt.Println`が`.Println`になり壊れるコードを生成する）。これはFR-7.21（別名除去）が要求する「宣言されている実際のパッケージ名を解決する」処理が今まで未実装だったことによるもので、第6回・第7回のテストは常に「無alias→有alias」方向だけだったため気づかれずに残っていた。`*types.PkgName.Imported().Name()`で実際の宣言パッケージ名（例:"fmt"）を取得し、無alias化の際はそちらを識別子名として使うよう修正した。
2. `internal/shape`に`NameVisibleAt(info, file, pos, name, except) bool`を新設。あるpositionにおいて、`name`という識別子がすでに(`except`以外の)何らかのオブジェクト（ローカル変数・別のimport・トップレベル宣言等）に束縛されているかを、`(*types.Scope).Innermost(pos)`から`Parent()`を辿ってpackage scopeまで（universe scopeの手前で打ち切り）順にチェックすることで判定する。
3. `internal/fix.ApplyToFile`に`collides(...)`チェックを追加：あるDecisionの適用先importについて、書き換え後の名前（`resolvedName`）が、そのimportの宣言位置および全ての使用箇所（`shape.SelectorIdentsOf`）のいずれかで、すでに別のオブジェクトに束縛されている場合はリネームをスキップする（trivial transformationの原則＝FR-7.5/FR-7.21の一般化）。
4. 3つのfixtureで検証：
   - `testdata/fix/collision_unaliased_to_alias`：`fmt`（無alias）を多数決で`f`にリネームしようとするが、同じ関数内にローカル変数`f`が存在 → **fixはスキップされ、ファイルは無変更**であることを確認。
   - `testdata/fix/collision_alias_to_unaliased`：`f "fmt"`を多数決で無alias化（`fmt`）しようとするが、同じ関数内にローカル変数`fmt`が存在 → **fixはスキップされ、ファイルは無変更**であることを確認。
   - `testdata/fix/no_collision_unrelated_scope`：`fmt`（無alias）を`f`にリネームする際、**別の**関数にローカル変数`f`が存在するが、そちらは`fmt`を一切参照していない → **fixは正常に適用され**、結果のファイルは「import別名`f`」と「無関係な関数内のローカル変数`f`」が共存する、普通に有効な(コンパイルが通る)Goコードになることを確認した。
5. 上記の衝突検出（`NameVisibleAt`）が実際に機能するには、型チェック時に`types.Info.Scopes`を明示的に（非nilマップとして）用意しておく必要があると分かり、既存の`fix_test.go`・`pipeline_test.go`にも`Scopes: map[ast.Node]*types.Scope{}`を追加した（後述の気づき参照）。

## 気づいたこと

1. **既存のシャドーイング（ユーザー指摘の(1)）は、すでに`shape.SelectorIdentsOf`の`types.Info.Uses`ベースの判定だけで正しく除外できていた**ことを、`no_collision_unrelated_scope`のgolden（fix後もローカル変数`f`と import別名`f`が共存する）で確認できた。Goの型チェッカーがスコープ規則を踏まえて`Uses`を解決済みのため、シャドーされた位置の識別子は元々`pkgName`には解決されない。つまり(1)は「今まで通りの実装で既に正しかった」ことの確認であり、コード変更は不要だった。
2. **`types.Info.Scopes`は他のフィールド（`Defs`/`Uses`/`Implicits`）と同様、非nilマップとして用意しないと型チェッカーが記録してくれない**、というgo/typesのAPIの落とし穴を実地で踏んだ。既存の2つのテストファイルでは`Scopes`を用意していなかったため、もし何もチェックせず先に衝突判定コードだけ書いていたら「衝突判定が常にfalseを返す（静かに無効化される）」というテストが検出しづらいバグになっていたはずで、危うく気づかずに進むところだった。CLI実装時（`go/packages`経由）も同様に、型情報のロードオプションで`Scopes`相当の情報が確実に含まれるようにする必要がある、という実装上の注意点として残る。
3. **衝突検出の実装（`Scope.Innermost(pos)`から`Parent()`を辿る）は、意図的に保守的（安全側）な近似になっている**：Goの変数スコープは本来「宣言以降、ブロック終端まで」だが、`types.Scope`はブロック内の全オブジェクトを（テキスト上の前後関係を区別せず）保持しているため、たとえ対象のimport使用箇所がローカル変数の宣言より**前**にあり、技術的には安全にリネームできるケースであっても、同じブロックに同名のローカル変数が存在するというだけで衝突と判定してしまう。これは「怪しければ直さない」というtrivial transformationの精神には合致しており、間違って壊れたコードを生成するよりは安全側に倒すべきだと判断したが、将来的により精密な「宣言前後」判定をすべきかは未検討のまま残した（→PRE-22）。
4. **`NameVisibleAt`はuniverse scope（`len`・`cap`等の組み込み識別子）との衝突は意図的にチェックしていない**。これも軽微だが、多数決の結果として例えば別名が`len`になるような極端なケースは理論上あり得るため、チェック範囲の境界として明文化しておく必要がある論点だと気づいた（→PRE-22であわせて整理）。

## 次にやること

- 上記の気づき3・4を`docs/01decision.pre.md`にPRE-22としてまとめて追記した。
- FR-7.21（別名除去）は今回、衝突検出も含めて実質的にカバーできた形になった。ただし「宣言パッケージ名の解決」と「衝突検出」は今回`internal/fix`単体のテスト（手組みのDecision）でのみ検証しており、`decide.Decide`が実際に`WantAlias == ""`を多数決の結果として選ぶケース（＝無aliasが多数派）をscan/decide経由のパイプラインテストではまだ確認していない。次のイテレーションで`internal/fix/pipeline_test.go`のようなE2Eテストにこのケースを追加できるとよい。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第9回：`docs/01decision.pre.md`をPRE-18〜22まで俯瞰し、古くなった内容を洗い出す

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: ユーザーから「`docs/01decision.pre.md`の論点を全体で俯瞰した時に、変化があるものはないか」という確認依頼を受けた実験。第6〜8回で書いたPRE-18〜22を、その後の実装（特に第7回・第8回）を踏まえて読み直し、内容が古くなっている・すでに実装で答えが出ている項目がないかを点検した。

## やったこと

1. PRE-18〜22を1件ずつ、「その後の実装で答えが変わっていないか」「推奨デフォルトがすでにコードとして実装済みでないか」の2軸で読み直した。
2. PRE-20（`astutil`と直接AST書き換えの使い分け）の「背景」に、第8回でFR-7.21（別名除去）を実装した際の知見と矛盾する記述が残っていることに気づいた：PRE-20は「FR-7.21はimport宣言の追加・削除が必要でastutilが要るだろう」と予想していたが、第8回の実装では**同一importパスのまま`spec.Name`を`nil`にするだけ**で成立しており、astutilは不要だった。ただし、この「成功パターン」自体は第8回時点では衝突ありのケース（`collision_alias_to_unaliased`、fixがスキップされる方）しかテストしておらず、「衝突なしで別名除去が実際に成功する」ケースを golden test で確認したことは一度もなかった、というテストの抜け穴にも気づいた。
3. 上記の抜け穴を埋めるため、`testdata/fix/unalias_no_collision/{input,golden}/main.go`（`f "fmt"` → 無alias`fmt`、衝突なし）を新設し、`internal/fix/collision_test.go`のテーブルに`alias_to_unaliased_no_collision`ケースを追加してgreenであることを確認した。これでPRE-20の訂正が実証付きの主張になった。
4. PRE-20の本文を、上記の訂正内容を反映する形に書き直した（当初の予想を残しつつ「第8回・第9回で判明した訂正」として追記）。astutilが必要になりそうな範囲は「FR-6.11（同一alias→複数import pathの解決）だけ」に狭まったことを明記した。
5. PRE-21・PRE-22も読み直したところ、内容自体は古くなっていなかったが、どちらも「推奨デフォルト」がすでに提案止まりではなく実装・テスト済みであるという事実が本文から読み取りにくかったため、それぞれに「現状」の一文を追記し、ユーザー確認さえ済めばそのままDECへ昇格できる状態であることを明記した。
6. PRE-18・PRE-19は読み直した結果、内容・ステータスとも変化なし（PRE-18＝推奨デフォルト通りinspector不使用のまま、PRE-19＝blank/dot importの除外は今も未実装のまま）と判断し、本文は変更しなかった。

## 気づいたこと

1. **PRE項目は「実装前に立てた予測」を含むため、後続の実装が進むと本文の一部が事実と食い違ってくる**ことがある、と気づいた。今回のPRE-20がまさにその例で、「これから実装するfixケースにはastutilが要るだろう」という予測が、実際に実装してみたら外れていた。ADR-00のループは「PRE→確認→DEC」という一方向の流れを想定しているが、実装を先行させる今回のような進め方（第6〜8回）では、確認前のPREの記述が実装によって上書きされるタイミングのズレが起きうる、という運用上の特性が見えた。
2. **「推奨デフォルトが実装済みかどうか」はPRE本文だけでは読み取れなかった**。PRE-21・PRE-22はどちらも「まだ提案段階」であるかのような書きぶりのままだったが、実際には第7回・第8回の時点でそれぞれ実装・テストまで完了しており、「ユーザー確認を待っているだけ」の状態だった。今後PREを書く際は、推奨デフォルトが（a）まだ未実装の提案なのか、（b）すでに実装・テスト済みで確認待ちなのかを、本文中に明示した方が、後から俯瞰した時に状況を誤解しにくいと分かった。
3. **「衝突ケース（失敗して当然）のテストはあるが、対応する成功ケースのテストがない」というテストの非対称性**は、PRE本文を読み直すという作業を通じて偶然見つかった。実装中（第8回）はバグ修正と衝突検出の実装に意識が向いており、「衝突検出が正しく動く」ことと「衝突が無ければ正しくfixが完了する」ことの両方を検証する必要がある、という基本的なテスト観点が漏れていた。

## 次にやること

- PRE-18〜22のうち、内容が実装済み・実証済みとなっているPRE-20〜22は、ユーザー確認が得られ次第まとめて`docs/01decision.md`のDECへ昇格できる状態にある。PRE-18・19は引き続き未実装のまま残っている。
- 気づき2を踏まえ、今後新たにPREを起票する際は「未実装の提案」か「実装・テスト済みで確認待ち」かを本文に明記する運用に変える。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第10回：PRE-18〜22をユーザーに確認し、DEC-11.18〜11.22として確定

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: 第9回で棚卸しした`docs/01decision.pre.md`のPRE-18〜22を、ADR-00のループ（PRE→ユーザー確認→DEC→本書へ知見追記）に従って1件ずつユーザーに確認し、`docs/01decision.md`のDEC-11.18〜DEC-11.22として確定した回。

## やったこと

1. PRE-18〜21をユーザーに確認したところ、いずれも推奨デフォルト通りで確定した：PRE-18（`internal/scan`は`inspector.Inspector`を使わず直接走査のまま）→DEC-11.18、PRE-19（blank/dot importをscan時点で除外）→DEC-11.19、PRE-20（同一import path内はdirect mutation、astutilはFR-6.11解決のみ）→DEC-11.20、PRE-21（`internal/scan`にもDEC-11.3を拡張）→DEC-11.21。
2. PRE-22（識別子衝突チェックの判定精度）は、**推奨デフォルト（現状の保守的な近似を恒久仕様として採用）ではなく**、ユーザー判断で「より精密な判定へ将来移行する」方針が選ばれた。第5回のPRE-17（多数決タイの表現、推奨デフォルトを退けて可変長型を選択）以来2度目の、推奨デフォルトからの逸脱。さらに「今回のセッションで実装まで進めるか、方針をDECとして記録するだけにして実装は別途にするか」を追加で確認したところ、**方針だけをDEC化し、具体的な実装（宣言前後の位置関係の区別・universe scopeとの衝突チェック）は別イテレーションに持ち越す**という判断だった。DEC-11.22はこの通り「方針決定・実装は保留」という、これまでのDECエントリにはなかった形で記録した。
3. `docs/01decision.md`をv1.6とし、DEC-11.18〜DEC-11.22を11節に追記、バージョン履歴にも要約を追加した。
4. `docs/01decision.pre.md`から、確定済みとなったPRE-18〜22を削除し、第5回（PRE-15〜17）の前例に倣って「確定済みで本書から削除した項目」の要約ブロックとして残した（未解決PRE項目0件の状態に戻した）。
5. DEC-11.19（blank/dot importの除外）はドキュメントの確定だけでなく実際のコード上の欠落だったため（`internal/scan.FromFiles`はこれまでblank/dot importをそのまま収集していた）、`internal/scan/scan.go`に除外処理を実装し、これまで存在しなかった`internal/scan`専用のテスト（`internal/scan/scan_test.go`）を新設して確認した。DEC-11.18・11.20・11.21はすでに実装・テスト済みの内容をそのまま追認する形だったため、コード変更は不要だった。DEC-11.22は方針のみの確定であるため、現行の`internal/shape.NameVisibleAt`実装（保守的な近似）はそのまま維持し、コード変更は行っていない。

## 気づいたこと

1. **PRE項目は「推奨デフォルト通り確定」と「推奨デフォルトを覆して確定」の両方がありうる**ことが、今回もPRE-22で再確認できた（第5回のPRE-17に続き2度目）。ADR-00のループが「推奨はあくまで提案であり拘束力がない」と明記している通りの運用になっている。
2. **DEC項目には「方針は確定するが実装は保留する」という種類のものがあってよい**、と今回のDEC-11.22で初めて明示的に扱った。これまでのDEC-11.1〜11.21はすべて「確定＝即座にコードへ反映済み（または反映不要）」だったが、PRE-22は「方向性には合意するが、コスト（宣言前後の位置関係を追う精密な解析の実装）をかけて今すぐやるかは別問題」というユーザー判断で、確定事項の中に「今後着手すべき保留タスク」を含める初めてのケースになった。今後DECを書く際は、この「確定＝実装済み」と「確定＝方針のみ、実装は別途」の2種類を区別して記録する必要があると分かった。
3. **DEC-11.19の確定は、ドキュメントの確定作業だけでは終わらず、実際のコード修正（と新規テスト）を要した**。これまでのラウンド（第6〜9回）では「先に実装する→後から確認してDEC化する」という順序だったが、PRE-19だけは「発見はしたが実装は先送りにしていた」項目だったため、今回のようにPRE確認の場で初めて実装するというケースが生じた。DECの確認作業とコードの実装作業が必ずしも同じタイミングで揃うとは限らない、という運用上の学びとして残す。

## 次にやること

- FR-6.11・FR-6.16・generated file skip・CLI本体（`cmd/goimportalias`）は引き続き未実装のまま。DEC-10.1の実装着手順（4節以降）に沿って次のイテレーションで着手できる。
- DEC-11.22（識別子衝突チェックの精密化）は、実装着手のタイミングが来たら`docs/fix-cases.md`のケース収集と合わせて対応する。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第11回：FR-6.16（単一ファイル内の重複 import）検出を実装（issue 03）

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: `docs/issues/03-fr6.16-detection.md`（依存なし・着手可能）を実装した回。第10回時点の「次にやること」に残っていたFR-6.16のうち、検出のみ（auto-fixはissue 10）を対象にした。

## やったこと

1. `internal/decide/decide.go`のpackage docが明示していた「FR-6.16はスコープ外」を解消し、`Decide`にFR-6.11（`byAlias`）と全く同じ形の第3の軸として`byFile map[fileKey][]shape.Occurrence`を追加。`fileKey{pkg, file, path}`でグルーピングし、同一グループ内でdistinct aliasが2つ以上あるものだけを`shape.DuplicateImport`として返すようにした（`Decide`の戻り値を3値`([]Decision, []AliasCollision, []DuplicateImport)`に変更）。
2. `internal/shape/model.go`の`Occurrence`に`File string`フィールドを追加し、`internal/scan/scan.go`で`isTest`と全く同じタイミング（`fset.Position(file.Pos()).Filename`）で埋めるようにした。`shape.DuplicateImport`型も新設。
3. `analyzer.go`に3番目の報告ループを追加し、`DuplicateImport`の全Occurrence（重複しているimport宣言行それぞれ）に対して`import %q is imported multiple times in this file with different aliases`を報告するようにした。
4. `Decide`のシグネチャ変更に伴い、既存の呼び出し側（`internal/decide/decide_test.go`・`internal/fix/pipeline_test.go`）を3値受け取りに更新。
5. `testdata/src/dupinfile/a.go`（1ファイル内で`"fmt"`を`f`/`g`の2 aliasでimport）を新設し、`analyzer_test.go`に`TestAnalyzer_DuplicateImportInSameFile`を追加。`internal/decide/decide_test.go`にも`TestDecide_DuplicateImports`をテーブル駆動で追加（同一ファイル内で検出／ファイル跨ぎでは不検出／同一alias2回は不検出、の3ケース）。

## 気づいたこと

1. **「file単位の情報はscanにしかない」という issue の前提は、Occurrenceにfilenameを1フィールド足すだけで解消できた**。`internal/scan.FromFiles`は元々`isTest`用に`fset.Position(file.Pos()).Filename`をper-file computeしていた（第7回で確認したisTestと同じパターン）ので、それをOccurrenceに載せて運ぶだけで、`internal/decide`側は追加のfsetを持つ必要も、`FromFiles`のシグネチャ（戻り値の数）を変える必要もなかった。結果として、FR-6.16の実際の検出ロジック（グルーピングと閾値判定）は、FR-6.11の`byAlias`と全く同じ形でdecide.go内に third axis として実装できた——scan/decideのどちらに置くか、という問いは「filenameというデータをどちらに置くか」と「グルーピングの判定ロジックをどちらに置くか」を分けて考えると、両方をdecideに寄せる方が既存のFR-6.11の実装パターンとの一貫性が高く、変更範囲も小さかった。
2. **同一alias・同一pathの重複importは有効なGoでは発生しない**（`f "fmt"; f "fmt"`は同一識別子の再宣言でコンパイルエラーになる）ため、`Decide`のdistinct alias判定はテストのために作った人工的なOccurrence列に対する防御的なガードに過ぎない。一方、`f "fmt"; g "fmt"`（同一path・異なるalias）は`go build`で実際にコンパイルが通ることを確認済み（一見冗長だが有効なGo）。
3. **既存のFR-6.10フィクスチャ（`dup`）が無変更のままgreenであること自体が、FR-6.16との非混同の実証になっている**。`analysistest`は期待していないdiagnosticsが出るとfailするため、`dup`パッケージ（複数ファイル・majority vote）が今回の変更後も新たなdiagnosticsを出さないことを確認できれば、それだけで「ファイル跨ぎの多数決」と「単一ファイル内の重複」が誤って混同されていないことの十分な証拠になる。issue 03の終了条件2番目はこの形で満たした。

## 次にやること

- FR-6.16のauto-fix（issue 10）は`internal/fix`側の作業として引き続き未着手。今回追加した`shape.DuplicateImport`（Occurrencesを全件保持）がそのまま入力になる想定。
- 残るD1系はissue 04（analyzer config・ignore・strict flag）のみ。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。

---

# 第12回：issue 04を題材にした「テスト実装フェーズ」の事前検討（コード変更なし）

- **日付**: 2026-07-21（同日）
- **本書の位置づけ**: 実際にissue 04（analyzer config・ignore・strict flag）を実装したわけではなく、「一つissueを選んで実装して」と指示された場合に、テストコードを書き始める前段階で何が事前に足りていないかを仮想的に検討した回。コード変更は行わず、ADR-02への追記（後述）のみ実施した。

## 気づいたこと

1. issue 04の本文は「必要なら`internal/shape/config.go`を作る」という書きぶりだが、実際には`File`/`Load`/`Save`/`Lookup`/`Merge`まですでに完成済みだった。`decide.Options.Strict`とその分岐（`decide.go`の`opts.Strict && len(distinct) > 1`）も実装済みだが、`decide_test.go`には対応するテストが1件もないことが`grep`で判明した。issueが一括起票された都合上、後続issueの土台が先行実装で前倒しに出来てしまうのは構造的に避けがたいと判断した。
2. `shape.File.Ignore`はフィールドとして存在するが、それを判定するヘルパー関数はコード全体に存在しない。`Lookup`内のprefix判定（`foo/...`）はinlineのままで、再利用可能な関数として切り出されていない。
3. `pass.Module`の実挙動のうち「`analysistest.Run`が`pass.Module.Dir`をどう設定するか」は未確認のまま残った（`go doc`で確認済みなのは"driverによってはnil"という一般論のみ、第2回気づき2）。issue 04の終了条件1番目（testdata配下に`importalias.json`を置くケース）が実際に成立するかは、この点の確認待ち。
4. `docs/issues/closed/`配下は`git log --stat`で確認した限り、移動された時の1コミットのみで、その後に「完了日・対応コミット」等が追記された実績は一度もない（ADR-01が許容している運用だが実際には使われていない、事実上の死んだ規定）。この事実を踏まえ、「関連issue（特にclosed）も読む」という手順は明示ルール化を見送った——README上でリンクされておらず、実際に読まれた実績もないため。
5. 上記1〜4を踏まえ、ADR-02に「実装中の非自明な発見は`docs/02notice.md`に追記する」手順（非自明の判定基準＝リトマス試験紙付き）を追加した。

## 悩み・保留にしたこと

- 「新しいテストを書く前に、対象の振る舞いがすでにテストされているか」の確認は`grep`で都度行うしかないが、毎回対象パッケージの既存テストファイルを全件スキャンするのはコストに見合わない場面もありそうで、線引きが定まっていない。今回は結論を出さず、次に同種の見落とし（テストの有無確認漏れ）が実際に問題を起こした際に改めて検討することにした。

## 次にやること

- 上記の悩みが実際に問題化したら、ADR-02の「現状把握」ステップ（手順2）に「対象パッケージの既存テストファイルの確認」を明示するかどうかを検討する。
- 保留：`02notice.md`自体の構成見直し（第2回から持ち越し、引き続き保留中）。
---

# 第5回: issue 05 vettool モード分岐の実挙動確認

- **日付**: 2026-07-21
- **やったこと**: `cmd/goimportalias` の vet ツールモード分岐を実装する際、`go vet -vettool=` が実際に起動する引数をラッパーで記録した。

## 気づいたこと

1. `go vet -vettool=` は `-flags`、`-V=full` に加えて、パッケージ解析時に `-json <workdir>/vet.cfg` という2引数形式で vettool を起動する。issue 05 と `docs/draft.md` のスケッチは「`*.cfg` を唯一の引数に取る規約」を前提にしていたが、Go 1.26 の実挙動では `-json` フラグが前置されるため、モード判定は `len(args)==3 && args[1]=="-json" && strings.HasSuffix(args[2], ".cfg")` も vet モードとして扱う必要がある。
