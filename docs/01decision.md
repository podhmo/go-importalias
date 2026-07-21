# importalias 実装方針決定書 (Implementation Decisions)

- **バージョン**: v1.6
- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/00origin.md`（機能仕様書、「何を提供するか」）を受けて、実装に着手するために必要な「どう実装するか」側の決定を、ユーザーとの確認を経て確定する。本書に記載の決定事項は、明示的な理由がない限りそのまま実装のベースラインとしてよい、自己完結した確定記録である。
- **v1.0からの変更点**: ユーザーとの確認の結果、D1（go vet analyzer）とD2（スタンドアロンCLI）を**単一バイナリ`goimportalias`**として提供する方針に変更した（1.3/1.4節）。これに伴い、OQ-5（FR-6.11ケースのauto-fix）もv1で対応する方針に変更した（3.1節）。
- **v1.1からの変更点**: `docs/draft.md`（リポジトリ全体のラフなドラフト実装）を書く中で見つかった、設計原則・パッケージレイヤ構成レベルの論点をユーザーと確認し、以下を確定した（詳細は11節）。
  - `internal/config`パッケージを廃止し、`internal/shape`に統合した（1.3節・2節）。scan/decide/fixが共有するドメイン型（`Occurrence`/`Decision`等）と、設定ファイルの型・読み書きを同一パッケージにまとめる。
  - D1（go vet analyzer）は`pass.Module`経由で`importalias.json`の読み込みを試みる（11.2節）。
  - `internal/shape`を除く`internal`配下のロジックパッケージは、標準入出力・ファイルシステムに直接触れない（11.3節）。
- **v1.3からの変更点**: `docs/draft.md`のドラフト作業で見つかった残りの設計・コーディング原則レベルの論点をユーザーと確認し、以下を確定した（詳細は11節）。
  - エラー表現は、CLI終了コード分岐に必要な箇所だけセンチネルエラーとし、他は`fmt.Errorf`でラップする（11.5節）。
  - `internal/fix`の書き換えは`astutil`＋`types.Info.Uses`＋`go/format.Node`とし、gofmt差分の混入を許容する（11.6節）。
  - CLIモードのマルチパッケージ処理はv1では逐次実行とする（11.7節）。
  - 設定ファイルの再生成は「保持する差分マージ」とする（11.8節）。
  - コード中のコメント・識別子は英語とする（設計文書・CLIメッセージは日本語可）（11.9節）。
  - テストは標準`testing`＋table-driven、`go-cmp`のみ例外許容、`testify`は不採用（11.10節）。
- **v1.4からの変更点**: v1.3の決定を`docs/draft.md`へ実際に反映する中で見つかった、より細かい適用粒度の論点（第3回）をユーザーと確認し、以下を確定した（詳細は11節）。
  - `shape.Occurrence.Alias`の無alias表現は空文字とする（11.11節）。
  - FR-6.11（同一alias→複数path）の検出結果は`shape.Decision`とは別の型・別スライスで返す（11.12節）。
  - `shape.Merge`は、スキャン範囲に含まれるpackageキーについてはpath単位も含め丸ごと新しい判定結果で置き換える（11.13節）。
  - `internal/fix`のSelectorExpr走査は1ファイル1回の事前インデックス化とする（11.14節）。
- **v1.5からの変更点**: `docs/draft.md`の`decideOne`（決定ロジック本体）を実際に書き下す中で見つかった論点（第5回）をユーザーと確認し、以下を確定した（詳細は11節）。
  - 設定ファイルのスコープ優先順位判定・パターンマッチのロジックは`internal/shape`（`File.Lookup`）に置き、複数`"foo/..."`マッチ時は最長prefixを優先する（11.15節）。
  - `AliasCollision`は`Occurrences []Occurrence`を持ち、診断位置を報告できるようにする（11.16節）。
  - 多数決タイの候補（`Decision.TieCandidate`・設定ファイルの`AliasValue.Tie`）は固定長2要素ではなく可変長（2要素以上）とし、DEC-2.1のスキーマをこの点で更新する（11.17節）。
- **v1.6からの変更点**: `go.mod`を実際に作成し、`internal/shape`・`internal/scan`・`internal/decide`・`internal/fix`・ルート`analyzer.go`を初めて生きたコードとして実装し、scan用（analysistest）・fix用（golden file）それぞれのテストハーネスを動かした一連の実験（`docs/02notice.md`第6〜9回）で見つかった論点をユーザーと確認し、以下を確定した（詳細は11節）。
  - `internal/scan`は`go/ast/inspector.Inspector`を使わず、`file.Imports`の直接走査のままとする（11.18節）。
  - blank import（`_`）・dot import（`.`）は`internal/scan`の時点で多数決・一貫性チェックの対象から除外する（11.19節）。
  - 同一import path内での別名変更（有alias⇄有alias・有alias⇄無alias）は常に直接AST書き換えとし、`astutil`はimport path自体の追加・削除を伴うケース（FR-6.11の解決）にのみ使う。DEC-11.6を狭める形で補足する（11.20節）。
  - `internal/scan`も`internal/fix`と同様、型情報（`*types.Info`）を呼び出し元からの明示・任意引数として受け取る設計とする。DEC-11.3の適用範囲を拡張する（11.21節）。
  - 識別子衝突チェック（FR-7.21）の判定精度について、より精密な判定（宣言前後の位置関係・universe scopeとの衝突）へ将来移行する方針をDECとして記録するが、具体的な実装は本書執筆時点では未着手のまま次回以降に持ち越す（11.22節）。
- **現時点で未確定の論点**: 11.22節（識別子衝突チェックのより精密な判定）は方針のみ確定しており、実装は今後のイテレーションに持ち越されている。それ以外に本書執筆時点で洗い出されていた実装上の論点はすべて本書内で確定済みである。テストスイートの構成・実行方法の細部（7節参照）のように、これ以上の言葉による決定ではなく実装しながらの実験によって詰めるべき事項については、10節（実装着手順）で最初のタスクとして明示する。

---

## 1. 基盤・環境

### 1.1 モジュールパス

- **DEC-1.1**: モジュールパスは `github.com/podhmo/go-importalias` とする。
  - 理由: ghqでの配置パス・ディレクトリ名と一致させ、`go get`/`go install`の取得パスに迷いを生じさせない。

### 1.2 Goバージョン

- **DEC-1.2**: `go.mod` の `go` ディレクティブは、ローカル開発環境の最新版に合わせ `go 1.26` とする。
  - 理由: ユーザー確認の結果、開発環境と揃え最新機能を使える状態を優先する方針とした。利用者環境での実行可能性（古いGoしかない環境）は許容し、必要になった時点でバージョン方針を見直す。

### 1.3 リポジトリのディレクトリレイアウト（OQ-1の解消）

- **DEC-1.3**: D1・D2を**単一バイナリ`goimportalias`**として提供する。`cmd/`配下はこのバイナリのみとし、それ以外のロジックはすべて`internal/`配下に置く。モジュールルート（`github.com/podhmo/go-importalias`、パッケージ名`importalias`）は、`go vet`・`golangci-lint`・その他の`analysis.Analyzer`ベースツール（multichecker等）から利用可能な`Analyzer`をexportするライブラリパッケージとする。

```
go.mod
analyzer.go            # package importalias（ルート）。var Analyzer = &analysis.Analyzer{...} をexport
cmd/
  goimportalias/        # 唯一のバイナリ。go vet -vettool= としても、CLIとしても動作する（1.4節）
    main.go
internal/
  shape/                # 5節・11.1節: 共有ドメイン型（Occurrence/Decision等）＋設定ファイルの型・読み書き・マージ
  decide/               # 4節: 決定ロジック（優先順位・多数決・tie）
  scan/                 # 6.5節: import走査・不整合検出ロジック（Analyzer実行とCLI実行の両方から共有）
  fix/                  # 7.3節・OQ-5: auto-fix（書き換え・識別子衝突チェック）
  genfile/              # 5.5節: generated file 判定
testdata/
  src/                  # analysistest用フィクスチャ（DEC-7.1）
  fix/                  # golden file方式のfixテスト用フィクスチャ（DEC-7.2）
docs/
  fix-cases.md          # testdata/fix/<case> と対応させたユースケース仕様書（DEC-7.2）
```

  - ルートパッケージは`internal`配下の`scan`/`decide`を呼び出す薄いAnalyzer定義のみを持ち、検出・決定ロジック自体は持たない（ロジックの実体は`internal/`に置くというOQ-1解消方針は維持）。
  - `internal/shape`の命名・責務範囲については11.1節を参照。

### 1.4 単一バイナリ`goimportalias`の起動モード分岐

- **DEC-1.4**: `cmd/goimportalias/main.go`は、起動時の引数を見て以下の2モードに自動分岐する。
  1. **vetツールモード**: `go vet -vettool=$(which goimportalias)`（FR-6.2）や、その他`unitchecker`プロトコル経由の呼び出しを検知した場合、`golang.org/x/tools/go/analysis/unitchecker` の `unitchecker.Main(importalias.Analyzer)` に処理を委譲する。
     - 検知方法: `unitchecker`が実際に用いる呼び出し規約（バージョン確認のための`-V=full`呼び出し、および設定用JSONファイルパスを唯一の引数として渡す規約）に合致するかで判定する。具体的な判定条件は実装時に`go vet -vettool=`の実挙動を見ながら固める（DEC-7.1のテストで実挙動を担保する）。
  2. **CLIモード**: 上記に該当しない場合、標準`flag`パッケージによるフラットなフラグ解析を行い、4節のCLI仕様（スキャン・設定ファイル生成・`-fix`）として動作する。
- **DEC-1.4.1（FR-6.15の維持）**: vetツールモードで動作する場合（`go vet`経由・golangci-lint等`Analyzer`変数を直接組み込む経由の両方を含む）は、ファイルシステムへの書き込みを一切行わない（設定ファイルの生成・更新、ソースコードの書き換えのいずれも不可）。書き込みを伴う処理（設定ファイル生成・`-fix`）はCLIモードでの直接実行時にのみ許可する。
  - 理由: origin.md FR-6.15の「analyzerは書き込みを一切行わない」という制約は、単一バイナリ化後も維持すべき安全上の要件である。go vetはunitcheckerプロトコル経由でパッケージ毎に別プロセス実行されるため、書き込みを許すと競合リスクが生じる（ADR-2と同じ理由）。

## 2. 設定ファイルの正確なスキーマ

origin.md 5.3/5.4節は「キー・値の正確な形は実装時に確定してよい」としていたため、以下の通り確定する。

### 2.1 Go型定義

- **DEC-2.1**: `internal/shape`（11.1節）に以下の型を定義する。

```go
package shape

type File struct {
	Packages map[string]map[string]AliasValue `json:"packages"`
	Ignore   []string                          `json:"ignore,omitempty"`
}

// AliasValue は次のいずれか一方の状態を表す:
//   - Resolved: 単一のalias文字列（""は「aliasなしが正」を意味する。FR-4.12）
//   - Tie:      len>=2の未解決候補（5.4節）。この間は自動修正の対象外（FR-4.10）
type AliasValue struct {
	Resolved string
	Tie      []string
}
```

  - JSON表現: `Resolved`が設定されている場合は文字列としてエンコード、`Tie`が設定されている場合は2要素以上の配列としてエンコードする（`MarshalJSON`/`UnmarshalJSON`をカスタム実装）。デコード時、文字列でも2要素以上の配列でもない値（要素数0や1の配列等）はエラーとする。
  - 理由: このJSON形式は一度公開すると利用者の設定ファイルに直接影響する外部フォーマットであるため、origin.md 5.3/5.4節のイメージ（`string | string[]`）に忠実な、読んで直感的にわかる形を優先した。
  - **DEC-11.17により更新**: 当初「ちょうど2要素」としていたが、多数決タイが3つ以上のaliasで発生するケース（FR-4.6は2択に限定していない）を表現できるよう、「2要素以上」に緩和した。

### 2.2 出力フォーマット規則

- **DEC-2.2**: 設定ファイルの自動生成・更新（FR-7.3）では、diffを安定させるため以下を規則とする。
  - インデントは半角スペース2つ（`json.MarshalIndent(v, "", "  ")`相当）、ファイル末尾は改行1つ。
  - `packages`内のスコープキー（`"*"` / `"foo/..."` / 明示パッケージ名）、および各スコープ内のimport pathキーは、出力時にアルファベット順にソートする。
  - `AliasValue.Tie`の2要素配列は、常に文字列として昇順ソートした状態で書き込む（出現順・多数決の順番には依存しない）。

### 2.3 generated file判定の正規表現

- **DEC-2.3**: 標準的な `^// Code generated .* DO NOT EDIT\.$`（[golang.org/s/generatedcode](https://golang.org/s/generatedcode) 準拠）を採用する。ファイル先頭のコメント帯（最初の非コメント・非空行が現れるまでの範囲）を走査し、マッチした場合に generated file と判定する。

### 2.4 FR-5.7「切り替えられること」の実現方法

- **DEC-2.4**: 設定ファイルのトップレベルは`packages`/`ignore`の2キーに固定されている（FR-5.9）ため、generated file判定のON/OFFは設定ファイルではなく実行時フラグで提供する（設定ファイルのスキーマは拡張しない）。
  - CLIモード: `-skip-generated`（bool, デフォルト `true`）
  - vetツールモード: analyzerフラグ `-importalias.skip_generated`（bool, デフォルト `true`。`go vet`の`-<analyzer>.<flag>`慣習に従う）

## 3. 決定ロジックの実装詳細

### 3.1 FR-6.11（同一alias・複数path）のauto-fix方針（OQ-5の解消）

- **DEC-3.1**: v1で**auto-fixを提供する**（`-fix`指定時のみ）。ヒューリスティックの詳細は、実装を進めながらユースケースごとに`testdata/fix/<case>`としてテストケースを追加し、それに対応する仕様を`docs/fix-cases.md`（DEC-7.2で新設）に明文化していく形で確定させる。`-fix`を指定しない限りは従来通り検出のみで、書き込みは発生しない。
  - 理由（v1.0からの変更）: 当初は「一般解がなく危険」として検出のみに留める案だったが、ユーザー判断により、具体的なユースケースをテストで積み上げながら安全な変換規則を都度確定させていく方針を採用した。FR-7.5「自明な変換」の定義（変数名衝突なし・セマンティクス不変）は本ケースにも適用され、この条件を満たさないケースは引き続きauto-fix対象外・検出のみとなる。

### 3.2 strictオプションの提供経路

- **DEC-3.2**: `strict`（FR-4.7）は設定ファイルには保存せず、実行時オプションとしてのみ提供する。
  - CLIモード: `-strict`（bool, デフォルト `false`）
  - vetツールモード: analyzerフラグ `-importalias.strict`（bool, デフォルト `false`）

## 4. CLIモードの実装詳細

### 4.1 フラグ一覧

- **DEC-4.1**: 以下のフラグを標準`flag`パッケージで定義する。サブコマンドは採用せず、フラットなフラグ構成とする（origin.md FR-7.17〜7.20の想定に一致）。対象パッケージは非フラグ引数（例: `./...`）で指定する。

| フラグ | 型 | デフォルト | 説明 |
|---|---|---|---|
| `-fix` | bool | `false` | auto-fixを適用する（FR-7.18、OQ-5ケースも含む） |
| `-config` | string | `""` | 設定ファイルパス。省略時はモジュールルート直下の`importalias.json`（FR-7.19/FR-5.8） |
| `-strict` | bool | `false` | 4.2節のstrictモード |
| `-skip-generated` | bool | `true` | generated fileをスキップする |

### 4.2 モジュールルート探索方法

- **DEC-4.2**: カレントディレクトリから親ディレクトリへ`go.mod`が見つかるまで辿る、シンプルな自前実装とする。`go/packages`や`go env GOMOD`の呼び出しには依存しない（軽量さ・依存の少なさを優先）。

### 4.3 FR-7.21の識別子解決手法

- **DEC-4.3**: `golang.org/x/tools/go/packages` を用いて型情報付き（`packages.NeedName | NeedTypes | NeedTypesInfo | NeedSyntax | NeedImports | NeedDeps`相当）でロードし、`go/types`の情報を用いて以下を行う。
  - 対象importの実際の宣言パッケージ名（`package`節の名前。import path末尾セグメントと不一致な場合に対応するため、FR-7.21で明示要求されている）の取得。
  - 書き換え後の識別子が、ファイル内の変数・他importのデフォルト識別子・その他スコープ内識別子と衝突しないことの意味的な確認。
  - 衝突する場合は当該ファイルをauto-fix対象外とし、診断のみ維持する（FR-7.21の要求通り）。

### 4.4 終了コード

- **DEC-4.4**: 以下の3値とする。
  - `0`: 不整合なし（もしくは`-fix`により全て解消）
  - `1`: 不整合を検出（tie未解決を含む。`-fix`の有無に関わらず、残存する不整合がある場合）
  - `2`: 実行時エラー（I/Oエラー・設定ファイルの構文不正・パッケージのロード失敗等）

### 4.5 出力・ログ

- **DEC-4.5**: 人間可読なプレーンテキストを標準出力に出す（FR-7.15）。「何を修正したか」「どこでtieが発生し保留されたか」を最低限含める。機械可読出力（`-json`等）はv1のスコープ外とし、必要になった時点で追加する。

## 5. Analyzer実体・golangci-lint連携

### 5.1 Analyzer変数の置き場所

- **DEC-5.1**: モジュールルートパッケージ（`package importalias`）に `var Analyzer = &analysis.Analyzer{Name: "importalias", ...}` を定義する。検出・決定ロジックの実体は`internal/scan`・`internal/decide`にあり、`Analyzer.Run`はそれらを呼び出す薄いラッパーとする。
  - `cmd/goimportalias/main.go`のvetツールモード（DEC-1.4）はこの`Analyzer`を`unitchecker.Main`に渡す。
  - golangci-lintのカスタムlinterとしての利用（FR-6.3）も、この`Analyzer`変数を直接importして参照する形で自明に満たされる。

### 5.2 多数決の集計単位（FR-6.17）

- **DEC-5.2**: テストパッケージ（`_test.go`、内部・外部いずれも）と非テストパッケージの出現回数は区別せず、合算して集計する。
  - 理由: origin.md自身が「実装上容易な方でよい」としている柔軟要件であり、実装を単純化する。

## 6. 依存ライブラリ方針

- **DEC-6.1**: 標準ライブラリに加えて `golang.org/x/tools`（`go/analysis`系一式、`go/analysis/unitchecker`、`go/analysis/passes/inspect`、`go/ast/inspector`、`go/packages`、`go/analysis/analysistest`）のみを外部依存とする。
  - v1.0からの変更点: D1/D2を単一バイナリ化したことに伴い、`go/analysis/singlechecker`ではなく`go/analysis/unitchecker`を採用する（`go vet -vettool=`が実際に要求するプロトコルは`unitchecker`側であるため）。
- **DEC-6.2**: CLIのフラグ解析は標準`flag`パッケージを用いる。cobra/pflag等の外部CLIフレームワークは採用しない。
  - 理由: `go vet`的な軽量ツールという性格に合わせ、依存を最小限に保つ。
- **DEC-6.3**: 設定ファイルのJSON処理は標準`encoding/json`のみを用いる。

## 7. テスト戦略の具体化（12節の実装）

以下の大枠は確定事項とするが、実行コマンド・ディレクトリ命名・golden更新手段などの細部は7.5節の通り、追加のヒアリングではなく最初の実装タスクでの試作・実験によって確定させる。

- **DEC-7.1**: Analyzer（scan側）のテストは `analysistest.Run` を用い、`testdata/src/<pkg>/...` 配下に入力ファイルと `// want` コメントによる期待診断を配置する標準パターンに従う。加えて、`go vet -vettool=`経由の実際の呼び出し（DEC-1.4の起動モード分岐）を検証する統合テストも用意する。
- **DEC-7.2**: auto-fix（`-fix`、fix側）のテストはgolden file方式とする。`testdata/fix/<case>/input/...` と `testdata/fix/<case>/golden/...` を用意し、実行結果をgoldenと比較する。各`<case>`について、対応するユースケース仕様を`docs/fix-cases.md`にケース名を一致させて記述し、「何のためのケースか」「どんな入力から何が期待されるか」を人間が読んでわかる状態にする（実装しながら都度追記する）。
- **DEC-7.3**: `internal/shape` の `AliasValue`（DEC-2.1）の `MarshalJSON`/`UnmarshalJSON` はtable-driven testで往復（round-trip）と異常系（不正な配列長など）を検証する。
- **DEC-7.4**: FR-12.2の通り、`testdata`ベースのテスト方式の確立を実装の最初のステップとする。

### 7.5 テストスイートの構成・実行方法の確定方針

- **DEC-7.5**: scan用（Analyzer/analysistest）・fix用（golden file）それぞれのテストスイートについて、以下の具体的な構成・実行方法はこれ以上のヒアリングでは決めず、最小限の仮実装（walking skeleton）を実際に書いて`go test ./...`を回しながら実験的に確定させる。これを実装全体の最初のタスクとする（10節DEC-10.1のステップ1〜3）。
  - 確定させる対象の例: golden再生成の具体的な手段（環境変数か専用フラグか等）、`testdata/fix/<case>`配下のファイル命名規則、scan側テストとfix側テストの実行コマンドの分け方（`go test ./internal/...`で両方回るようにするか等）、CIワークフロー（DEC-8.1）から見た実行方法。
  - 理由: この部分は「正解を知っていて選ぶ」種類の決定ではなく、実際に手を動かして初めて使い勝手や過不足が判断できる領域であるため、プロトタイピングを通じて決める方が確実である。ここが固まれば、以降は「テストケースを追加してから実装する」というループだけで機能拡充を継続できる。

## 8. CI・配布・ライセンス

- **DEC-8.1**: GitHub Actionsで `go build ./...` / `go vet ./...` / `go test ./...` を実行するワークフローを用意する。
- **DEC-8.2**: v1の配布は `go install github.com/podhmo/go-importalias/cmd/goimportalias@latest` のみとし、goreleaser等による自動リリース・バイナリ配布はv1スコープ外とする。
- **DEC-8.3**: ライセンスはMITとする。
- **DEC-8.4**（OQ-2の解消）: `github.com/podhmo/go-importalias` として公開する前提で進める。

## 9. 明示的にスコープ外へ送る論点（次フェーズ送り）

- **OQ-4**（go.work下の「自モジュール」判定）: v1では標準ツールチェーンの挙動に任せ、独自対応は行わない。
- **OQ-6**（workspace横断の共有設定ファイル）: 引き続き未検討のまま次フェーズへ送る。設定ファイルはモジュールルート単位（FR-5.1）のままとする。
- OQ-5は3.1節DEC-3.1の通りv1スコープ内に取り込まれたため、本節からは除外した（実装しながら`docs/fix-cases.md`で仕様を確定させていく）。

## 10. 実装着手順（OQ-3の解消）

- **DEC-10.1**: 以下の順序で着手する。最優先タスクは1〜3の「テストスイート（scan用・fix用）の構成・実行方法を仮実装で確立すること」であり（DEC-7.5）、これが完了すればそれ以降は「テストケースを追加してから実装する」というループの繰り返しで機能拡充を継続できる。
  1. `internal/shape`: DEC-2.1の型定義（共有ドメイン型＋設定ファイル型）と(Un)MarshalJSON、DEC-7.3のround-tripテスト。fixのgolden側テストが設定ファイルを扱うために必要な最小限を用意する。
  2. **scan側テストハーネスの確立**: 何も診断しない空のAnalyzerと`testdata/src`配下の最小フィクスチャを用意し、`analysistest.Run`が実際に動く状態を作る（DEC-7.1・DEC-7.4）。
  3. **fix側テストハーネスの確立**: 何も書き換えないCLI（`cmd/goimportalias`の最小実装）と`testdata/fix/<case>`（input==golden の最小ケース）を用意し、golden比較が実際に動く状態を作る（DEC-7.2）。ここまでで2〜3の「テストスイートの構成・実行方法」が確定したものとする。
  4. `internal/scan`・`internal/decide`: 6.5節の検出ロジックと4節の決定ロジック（多数決・優先順位・tie判定）を、2で確立したテストハーネス上にテストケースを追加しながら実装する。
  5. ルートパッケージの`Analyzer`実体化（診断のみ）＋`cmd/goimportalias`のvetツールモード（DEC-1.4、書き込みなし）。
  6. `internal/genfile`: generated file判定。
  7. `internal/fix` + `cmd/goimportalias`のCLIモード: スキャン・設定ファイル生成・auto-fix（OQ-5ケースを含め、3で確立したテストハーネス上に`testdata/fix/<case>`と`docs/fix-cases.md`を都度追加しながら実装する）。
  - 理由: テストハーネスを先に動く状態にしておくことで、以降の各機能はハーネスの上に「ケース追加→実装」を繰り返すだけで安全に積み上がる。

## 11. 設計原則・レイヤ構成の確定（`docs/draft.md`のドラフト作業からの追補）

`docs/draft.md`（リポジトリ全体をラフに1ファイルへスケッチしたドラフト）を書く中で見つかった、個々のパッケージの実装詳細ではなく**設計原則・パッケージレイヤ構成レベル**の論点について、ユーザーとの確認を経て以下を確定する。

### 11.1 共有ドメイン型・設定ファイル型の置き場所

- **DEC-11.1**: `internal/config`パッケージは廃止し、`internal/shape`に統合する。`scan`が集める1件のimport出現・`decide`が下す1件の判定を表す共有ドメイン型（`Occurrence`・`Decision`等）と、設定ファイル（`importalias.json`）の型・読み書き（DEC-2.1の`File`/`AliasValue`、`Load`/`Save`）を、同一パッケージ`internal/shape`にまとめる。パッケージ内はファイルで役割を分けてよい（例: `shape/model.go`にドメイン型、`shape/config.go`に設定ファイル関連）。
  - 理由: 設定ファイルの型・読み書きはコード量として小さく、かつ「複数のパッケージ（`scan`/`decide`/`fix`/`cmd`）から共有して参照される型」という性質はドメイン型と共通しているため、無理に`internal/model`と`internal/config`を分けず、依存パッケージが1つで済むようにする。ユーザー判断により、当初案の`internal/model`という名称も採用せず、`internal/shape`という名称を正式採用した。
  - 1.3節のディレクトリレイアウトはこの決定に合わせて更新済み。

### 11.2 D1（go vet analyzer）による設定ファイルの読み込み

- **DEC-11.2**: D1（go vet analyzer）は、`pass.Module`（`golang.org/x/tools/go/analysis`の`Pass.Module`フィールド）経由でモジュールルートを推定し、`importalias.json`の読み込みを試みる。
  - `pass.Module`が`nil`である、モジュールルートを特定できない、またはファイルが存在しない等で読み込みに失敗した場合は、その旨をstderrにログ出力した上で、「設定ファイルなし」として動作を継続する（4.2節の多数決のみで判定し、`ignore`は空として扱う）。
  - 理由: `ignore`（5.7節）によるスキップ判定はCIとしての誤検出（本来チェック対象外のパッケージへの誤診断）を避けるために重要であり、D1が設定ファイルの内容を一切無視するのは実用上望ましくないと判断した。設定ファイルの読み込みはread-onlyな操作でありFR-6.15（D1はファイルシステムへの書き込みを一切行わない）には抵触しない。
  - `pass.Module`の信頼性（ドライバによってはnilになりうる: `go doc`上の記述で確認済み）は環境依存のリスクとして残るが、読み込みに失敗した場合の扱い（stderrログ＋設定ファイルなし相当への自動フォールバック）を明確にすることで、環境差によってD1の診断結果が「クラッシュする」「原因不明に挙動が変わる」事態を避ける。
  - `analysis.Analyzer`の`Requires`フィールド（`golang.org/x/tools/go/analysis`で実在確認済み。`inspect.Analyzer`が同じ機構で使われている）を用いて、設定ファイルの読み込みを行う内部的なサブAnalyzerに切り出してもよい（同一パス内での重複読み込みを避けられる）。ただしこれは実装上の整理術であり、上記の「読み込み失敗時の扱い」という原則自体を変えるものではない。

### 11.3 `internal/fix`が識別子衝突チェック（FR-7.21）に使う型情報の受け渡し方

- **DEC-11.3**: `internal/fix`の公開APIは、識別子衝突チェック（FR-7.21）・qualified identifierの安全な書き換え（FR-7.7）に必要な型情報（`*types.Info`・`*types.Package`）を、呼び出し元から明示的な引数として受け取る形にする。`golang.org/x/tools/go/packages`によるロードは、CLIモード（`cmd/goimportalias`）だけの責務とし、`internal/fix`自体は`go/packages`に依存しない。
  - 理由: 型情報のロード（重い処理になりうる）をロジックパッケージの内部に隠さず、最上位のエントリポイントだけの責務にすることで、`internal/fix`を「受け取ったAST・型情報に対する純粋な変換」として単体テストしやすくする（11.4節の副作用境界の原則と一貫させる）。

### 11.4 `internal`配下のロジックパッケージの副作用境界

- **DEC-11.4**: `internal/shape`（設定ファイルの読み書きが責務そのものである部分）を除き、`internal/scan`・`internal/decide`・`internal/fix`・`internal/genfile`は標準入出力・ファイルシステムに直接触れない。これらのパッケージは受け取ったデータ（AST・型情報・`shape`の型等）に対する変換を行い、結果をデータとして返すだけの純粋な関数群として実装する。
  - 人間向けの出力整形（FR-7.15: 「何を修正したか」「どこでtieが発生し保留されたか」等の表示）は`cmd/goimportalias`だけが行う。
  - 理由: golden fileテスト・table-driven testを書きやすくするため。副作用を持つのは「その副作用自体が責務であるパッケージ」（`internal/shape`の設定ファイルI/O）だけに限定する。

### 11.5 `internal/scan`・`internal/decide`・`internal/fix`のエラー表現

- **DEC-11.5**: CLIの終了コード分岐（DEC-4.4）で「実行時エラー」かどうかを`errors.Is`で区別する必要がある箇所だけ、センチネルエラー（`var ErrXxx = errors.New(...)`）を各パッケージで定義する。それ以外は通常の`fmt.Errorf("...: %w", err)`でラップする。エラー文字列は英語・小文字始まりとし、`"importalias: "`のような固定プレフィックスは付けない（文脈は`cmd/goimportalias`側で一箇所にまとめて付加する）。
  - 理由: プレフィックスを各所で手打ちすると表記揺れが起きやすく、呼び出し元でまとめて文脈を付加する方が保守しやすい。センチネルエラーもDEC-4.4の判定に本当に必要な箇所に限定し、過剰な定義を避ける。

### 11.6 `internal/fix`のimport節・識別子書き換えの実装方針

- **DEC-11.6**: import節の追加・削除は`golang.org/x/tools/go/ast/astutil`の`AddNamedImport`/`DeleteNamedImport`を使う。qualified identifier（`pkg.Symbol`）の書き換えは、`go/ast.Inspect`で`*ast.SelectorExpr`を走査し、`*types.Info.Uses`（DEC-11.3で受け取る型情報）を使って本当にそのimportを指しているか確認した上で行う（識別子の文字列一致だけに頼らない）。書き戻しは`go/format.Node`を使い、結果がgofmt整形された状態になることは許容する。
  - 理由: 対象ファイルは自明な変換（FR-7.5）が入る時点でそもそも書き換えが発生するため、gofmt差分が混ざること自体は許容範囲とみなす。

### 11.7 CLIモード（D2）でのマルチパッケージ処理

- **DEC-11.7**: v1では逐次実行とする。
  - 理由: (1) golden fileテスト（DEC-7.2）の出力順序を決定的にしたい、(2) 設定ファイルへの書き込みが1箇所に集約されるため、並行化すると書き込み競合の考慮が必要になり複雑さが増す、(3) 軽量ツールという性格（DEC-6.2）に合わせ、まずは単純さを優先する。パフォーマンスが実際に問題になった時点で`errgroup`等の導入を検討する。

### 11.8 設定ファイルの再生成（FR-7.3）のマージ方針

- **DEC-11.8**: 「保持する差分マージ」を採用する。今回のスキャンで見つかったpackage/pathの組は新しい判定結果で上書きし、スキャン範囲に含まれないpackage/pathの組の既存エントリはそのまま残す。
  - 理由: 部分スコープでのスキャン（FR-7.13）を安全に行うためには、スキャン対象外の既存設定を意図せず消してしまわない方が事故が少ない。「毎回完全に再構築」する代替案は、部分スコープ運用と相性が悪い。

### 11.9 コード中のコメント・識別子の言語

- **DEC-11.9**: ソースコード中の識別子・コメントは英語とする。設計文書（`docs/*.md`）・CLIの人間向け出力メッセージ（FR-7.15）は日本語を許容する。
  - 理由: DEC-8.4で`github.com/podhmo/go-importalias`として公開する前提と確定しており、公開Goライブラリとしての一般的な慣習（英語コメント・`godoc`規約）に合わせる方が、後から翻訳し直すコストを避けられる。

### 11.10 テストコードの依存方針

- **DEC-11.10**: テストコードもDEC-6.1〜6.3と同じ最小依存方針に含める。標準`testing`パッケージ＋`t.Run`によるtable-driven testを基本とし、比較には`reflect.DeepEqual`、または構造体の深い比較（AST・トークン位置を含む）が不便な場合に限り`github.com/google/go-cmp`をテスト専用の例外的な追加依存として許容する。`testify`は導入しない。
  - 理由: 依存を最小限に保つという方針をテストにも一貫させる。ただし`go-cmp`はテスト専用であればモジュールの実行時依存グラフを汚さないため、深い構造体比較の実用性を優先して例外とする。

### 11.11 `shape.Occurrence.Alias`の「無alias」表現

- **DEC-11.11**: `shape.Occurrence.Alias`は、明示的なaliasが書かれていないimportの場合は空文字（`""`）とする。実際の宣言パッケージ名（無aliasの場合の「暗黙のalias」）が必要な箇所は、`go/types`から都度解決する。
  - 理由: DEC-2.1・FR-4.12がすでに設定ファイルの`AliasValue.Resolved`について「`""`は`aliasなしが正`を意味する」と定めており、`scan`が生成し`decide`が参照する`Occurrence.Alias`も同じ規約（空文字＝無alias）に揃える方が一貫し、二重の規約を持たずに済む。

### 11.12 FR-6.11（同一alias・複数path）の検出結果の型

- **DEC-11.12**: `internal/decide`は、FR-6.10（同一path→複数alias、`shape.Decision`が担う）とFR-6.11（同一alias→複数path）を別軸の検出結果として扱う。後者は`shape.Decision`には持たせず、`shape.AliasCollision{Package, Alias string; Paths []string}`のような別型・別スライスとして返す（例: `decide.Decide`は`([]shape.Decision, []shape.AliasCollision, error)`を返す）。
  - 理由: `Decision`は「(Package, Path)の組について何のaliasを使うべきか」という1軸の判定を表す型として単純に保ち、「1つのaliasに複数pathが対応している」という別軸の検出結果を無理に同じ型に押し込まない方が、それぞれの型の責務が明確になる。

### 11.13 `shape.Merge`の適用粒度

- **DEC-11.13**: `shape.Merge`は、今回のスキャン範囲に含まれるpackageキーについては、そのpackage内のpathエントリも含めて新しい判定結果（fresh）で丸ごと置き換える（path単位での差分保持はしない）。スキャン範囲に含まれないpackageキーはDEC-11.8の通りそのまま残す。
  - 理由: 設定ファイル（`importalias.json`）は本来「今スキャン範囲内で本当に必要な内容だけが載っている」状態が理想であり、使われなくなったimportの古いtie記録・alias指定を残し続けるのは望ましくない。一時的な検出漏れ（ビルドタグで除外されたファイル等）で設定が失われるリスクよりも、設定ファイルを可能な限り最小限に保つことを優先する。`ignore`（5.7節）は本決定の対象外（package/pathのマージとは別の扱いのため、DEC-11.8同様スキャン結果によらず保持される）。

### 11.14 `internal/fix`のSelectorExpr走査の設計

- **DEC-11.14**: `internal/fix`は、SelectorExprの走査を1ファイルにつき1回だけ行い、`file.Imports`と`typesInfo.Uses`から「importの束縛先（`*types.PkgName`）ごとのSelectorExprリスト」を事前にインデックス化する（例: `map[*types.PkgName][]*ast.SelectorExpr`）。各`Decision`の適用ループでは、このインデックスを引くだけにする。
  - 理由: 素朴に`Decision`ごとに毎回ファイル全体のSelectorExprを走査すると、1ファイルに複数の書き換え対象importがある場合にDecision数×ファイルサイズのオーダーになる。事前インデックス化すれば線形で済み、「収集」と「適用」の責務も分離できて実装の見通しがよい。

### 11.15 設定ファイルのスコープ優先順位判定（FR-4.1〜4.3）の置き場所

- **DEC-11.15**: 優先順位判定（明示パッケージ名 > `"foo/..."` > `"*"`）・パターンマッチのロジックは`internal/shape`に持たせる。`File`に`func (f *File) Lookup(pkg, path string) (AliasValue, bool)`を生やし、`internal/decide`はこのメソッドを呼ぶだけにする（優先順位判定自体を知らなくてよい）。
  - あわせて、複数の`"foo/..."`スコープが同時にマッチする場合（例: `"foo/..."`と`"foo/bar/..."`がどちらも`"foo/bar/baz"`にマッチ）は、最長prefixを優先する。
  - 理由: 設定ファイルの生データ形式（`Packages map[string]map[string]AliasValue`とスコープキーの意味）を知っているのは`shape`パッケージだけであるべきで、`decide`はpath方向/alias方向の集計とタイ判定という決定ロジック本体に専念させる。最長prefix優先は、より具体的な指定（対象範囲が狭い方）を優先するという直感に合う規則として採用した。

### 11.16 `AliasCollision`（FR-6.11の検出結果）の診断位置情報

- **DEC-11.16**: `shape.AliasCollision`の`Paths []string`を`Occurrences []Occurrence`（各pathにつき最初に見つかった1件の`Occurrence`を代表として保持）に置き換える。`analyzer.go`は`c.Occurrences[0].Pos`を使って先頭の代表位置に診断を出す。
  - 理由: 第3回時点の型は位置情報を一切持たず、`analyzer.go`側で`pass.Reportf`を呼ぶ手段がなかった。既存の`Occurrence`型（`Pos`を持つ）を再利用すれば新しい位置情報付き型を増やさずに済む。

### 11.17 多数決タイが3つ以上のaliasで発生した場合の表現

- **DEC-11.17**: `shape.Decision.TieCandidate`を固定長`[2]string`ではなく可変長`[]string`（2要素以上、アルファベット順）に変更する。あわせて、設定ファイル側の`AliasValue.Tie`（DEC-2.1）の制約も「ちょうど2要素」から「2要素以上」に変更し、`MarshalJSON`/`UnmarshalJSON`は2要素未満の場合にのみエラーとする。
  - 理由（推奨デフォルトからの変更）: 実装者側の推奨は「型は`[2]string`のまま変更せず、3つ以上タイの場合は先頭2件のみ記録する」だったが、ユーザー判断により型そのものを可変長にして3つ以上の同数タイも欠落なく表現する方針を採用した。`docs/00origin.md` FR-4.6は「最多の組が複数存在する場合は自動確定しない」としか定めておらず2択に限定していないため、型として表現しきれない情報を切り捨てるより、可変長にして実際に起きたタイをそのまま保持する方が安全と判断した。
  - 本決定はDEC-2.1（`AliasValue`のスキーマ）を「2要素固定」から「2要素以上」へ変更する形で上書きする。

### 11.18 `internal/scan`は`inspector.Inspector`を使わない

- **DEC-11.18**: `internal/scan`は`go/ast/inspector.Inspector`を使わず、`file.Imports`を直接走査する。ルート`analyzer.go`の`Analyzer.Requires`は空のままとし、`inspect.Analyzer`への依存は持たせない。
  - 理由: `docs/draft.md`時点では`inspector.Inspector`を使う設計だったが、「使う理由が言語化されていない」という保留（第2回）があった。実際にimport宣言のみを対象とする最小実装（`docs/02notice.md`第6回）で`analysistest`によるFR-6.10検知が問題なく成立し、`Requires`を空にできることが実証された。他のAnalyzerとの計算共有や、generated file判定等でAST全体を舐める処理が将来具体化した場合は、その時点で`inspector.Inspector`導入を再検討する。DEC-6.1が依存として列挙している`go/analysis/passes/inspect`・`go/ast/inspector`は、現時点では未使用のまま「許容されてはいるが使っていない」依存として扱う。

### 11.19 blank import（`_`）・dot import（`.`）の扱い

- **DEC-11.19**: `internal/scan`は、`Alias`が`"_"`（blank import）または`"."`（dot import）である occurrence を、多数決・一貫性チェックの対象から除外する。
  - 理由: これらは通常の「別名の揺れ」の議論の対象ではなく、`internal/decide`の多数決候補にそのまま混入すると誤った"correct"判定を生みかねない（`docs/02notice.md`第6回で発見）。`docs/00origin.md`・本書のいずれにもこれまで扱いの記載がなかったため、ここで明文化する。

### 11.20 同一import path内の別名変更は常に直接AST書き換え（DEC-11.6の補足）

- **DEC-11.20**: 同一import path内での別名変更（有alias⇄有alias、および有alias⇄無alias＝FR-7.21の別名除去を含む）は、常に既存`*ast.ImportSpec.Name`の直接書き換えで行う。`astutil.AddNamedImport`/`DeleteNamedImport`は、import path自体の追加・削除を伴うケース（FR-6.11の解決。本書執筆時点で未実装）にのみ用いる。
  - 理由: DEC-11.6は当初astutilの使用を前提としていたが、`docs/02notice.md`第8回・第9回で、FR-7.21（別名除去）を含む同一パス内の別名変更が、import宣言の追加・削除なしに`spec.Name`の書き換えだけで成立することを、実際にgolden testで確認した（`testdata/fix/unalias_no_collision`ほか）。astutilが必要になりそうな範囲は、これによりFR-6.11の解決だけに狭まった。DEC-11.6自体は「import path自体の追加・削除」の方針として引き続き有効であり、本決定はその適用範囲を明確化する補足という位置づけ。

### 11.21 `internal/scan`も型情報を明示引数として受け取る（DEC-11.3の拡張）

- **DEC-11.21**: DEC-11.3の原則（型情報は呼び出し元からの明示引数として受け取り、パッケージ自身は`go/packages`を呼ばない）を、`internal/fix`だけでなく`internal/scan`にも明文で拡張する。`internal/scan.FromFiles`は`*types.Info`を任意（nil許容）の引数として受け取り、渡された場合のみ各importの使用箇所位置（`shape.Occurrence.UsePos`）を解決する。型情報のロードは引き続きエントリポイントの責務とし、go vet Analyzer（D1）は`pass.TypesInfo`を driver から無償で受け取れるためこの点で追加実装は不要、CLI（D2）は`internal/fix`向けと同様に`go/packages`で型情報付きロードを行う（DEC-4.3の対象を`internal/scan`にも広げる）。
  - 理由: `docs/02notice.md`第7回で、go vetの診断位置を「実際の使用箇所（fix対象）」にするために`internal/scan`にも型情報が必要になることが判明し、実装・テストまで完了した。型情報がnilの場合`UsePos`は空のまま返り、診断はimport宣言行にフォールバックするため、型情報の有無どちらの文脈でも同じAPIで動作する。

### 11.22 識別子衝突チェック（FR-7.21）の判定精度の将来方針

- **DEC-11.22**: 現在の識別子衝突チェック（`internal/shape.NameVisibleAt`）は、意図的に保守的な近似のままとする：(a) 同一ブロック内であれば、対象の使用箇所がローカル変数の宣言より前にあり技術的には安全な場合でも「同名のローカル変数が同じブロックに存在する」というだけで衝突と判定する（宣言前後の位置関係は区別しない）。(b) universe scope（`len`・`cap`等の組み込み識別子）との衝突は判定対象外とする。将来的には、この2点についてより精密な判定（宣言前後の位置関係を区別する判定、universe scopeとの衝突チェック）へ移行する方針とする。
  - **現状**: 本決定は方針の確定のみであり、より精密な判定の具体的な実装は本書執筆時点では着手していない（現行の保守的な実装のまま据え置き）。実装は、それが原因でfixが不必要にスキップされる事例が実際に`docs/fix-cases.md`のケース収集の中で出てきた場合、またはユーザーが着手を指示した場合に、別イテレーションで行う。
  - 理由: ユーザー判断により、実装者側の推奨（現状の保守的な近似を正式仕様として恒久的に採用する）ではなく、より精密な判定を将来目指す方針を採用した。ただし今回のセッションでは方針の記録に留め、実装コストは別スコープとする判断も合わせて示された。
