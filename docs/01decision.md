# importalias 実装方針決定書 (Implementation Decisions)

- **バージョン**: v1.1
- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/00origin.md`（機能仕様書、「何を提供するか」）を受けて、実装に着手するために必要な「どう実装するか」側の決定を、ユーザーとの確認を経て確定する。本書に記載の決定事項は、明示的な理由がない限りそのまま実装のベースラインとしてよい、自己完結した確定記録である。
- **v1.0からの変更点**: ユーザーとの確認の結果、D1（go vet analyzer）とD2（スタンドアロンCLI）を**単一バイナリ`goimportalias`**として提供する方針に変更した（1.3/1.4節）。これに伴い、OQ-5（FR-6.11ケースのauto-fix）もv1で対応する方針に変更した（3.1節）。
- **現時点で未確定の論点**: 本書執筆時点で洗い出されていた実装上の論点はすべて本書内で確定済みである。テストスイートの構成・実行方法の細部（7節参照）のように、これ以上の言葉による決定ではなく実装しながらの実験によって詰めるべき事項については、10節（実装着手順）で最初のタスクとして明示する。

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
  config/               # 5節: 設定ファイルの型・読み書き・マージ
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

- **DEC-2.1**: `internal/config` に以下の型を定義する。

```go
package config

type File struct {
	Packages map[string]map[string]AliasValue `json:"packages"`
	Ignore   []string                          `json:"ignore,omitempty"`
}

// AliasValue は次のいずれか一方の状態を表す:
//   - Resolved: 単一のalias文字列（""は「aliasなしが正」を意味する。FR-4.12）
//   - Tie:      len==2の未解決候補（5.4節）。この間は自動修正の対象外（FR-4.10）
type AliasValue struct {
	Resolved string
	Tie      []string
}
```

  - JSON表現: `Resolved`が設定されている場合は文字列としてエンコード、`Tie`が設定されている場合は2要素配列としてエンコードする（`MarshalJSON`/`UnmarshalJSON`をカスタム実装）。デコード時、文字列でも2要素配列でもない値（要素数1や3以上の配列等）はエラーとする。
  - 理由: このJSON形式は一度公開すると利用者の設定ファイルに直接影響する外部フォーマットであるため、origin.md 5.3/5.4節のイメージ（`string | string[]`）に忠実な、読んで直感的にわかる形を優先した。

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
- **DEC-7.3**: `internal/config` の `AliasValue`（DEC-2.1）の `MarshalJSON`/`UnmarshalJSON` はtable-driven testで往復（round-trip）と異常系（不正な配列長など）を検証する。
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
  1. `internal/config`: DEC-2.1の型定義と(Un)MarshalJSON、DEC-7.3のround-tripテスト。fixのgolden側テストが設定ファイルを扱うために必要な最小限を用意する。
  2. **scan側テストハーネスの確立**: 何も診断しない空のAnalyzerと`testdata/src`配下の最小フィクスチャを用意し、`analysistest.Run`が実際に動く状態を作る（DEC-7.1・DEC-7.4）。
  3. **fix側テストハーネスの確立**: 何も書き換えないCLI（`cmd/goimportalias`の最小実装）と`testdata/fix/<case>`（input==golden の最小ケース）を用意し、golden比較が実際に動く状態を作る（DEC-7.2）。ここまでで2〜3の「テストスイートの構成・実行方法」が確定したものとする。
  4. `internal/scan`・`internal/decide`: 6.5節の検出ロジックと4節の決定ロジック（多数決・優先順位・tie判定）を、2で確立したテストハーネス上にテストケースを追加しながら実装する。
  5. ルートパッケージの`Analyzer`実体化（診断のみ）＋`cmd/goimportalias`のvetツールモード（DEC-1.4、書き込みなし）。
  6. `internal/genfile`: generated file判定。
  7. `internal/fix` + `cmd/goimportalias`のCLIモード: スキャン・設定ファイル生成・auto-fix（OQ-5ケースを含め、3で確立したテストハーネス上に`testdata/fix/<case>`と`docs/fix-cases.md`を都度追加しながら実装する）。
  - 理由: テストハーネスを先に動く状態にしておくことで、以降の各機能はハーネスの上に「ケース追加→実装」を繰り返すだけで安全に積み上がる。
