# importalias 実装決定 ドラフト (Pre-Decision Draft)

- **バージョン**: v0.1 Draft
- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/00origin.md`（機能仕様書）は意図的に「どう実装するか」を範囲外としている。本書はその「How」側で実際に手を動かすために確定させる必要がある論点を洗い出し、それぞれについてデフォルト案（採用候補）を添えて列挙する作業用ドラフトである。
- **本書の使い方**: 各論点について「デフォルト案」を仮採用しつつ`docs/01decision.md`へ確定事項として清書する。ここでの記述は最終決定ではなく検討途中のメモ。

---

## 1. 基盤・環境

### 1.1 モジュールパス
- 現状: ディレクトリ名 `go-importalias`、ghq配置 `github.com/podhmo/go-importalias`、git user `podhmo`。
- 選択肢:
  - A. `github.com/podhmo/go-importalias`（ディレクトリ・ghq配置に一致）
  - B. `github.com/podhmo/importalias`（プロジェクト名`importalias`に一致、リポジトリ名だけ`go-`prefix）
- デフォルト案: **A**。ghqの配置と一致させておくと`go get`等の取得パスに迷いが生じない。

### 1.2 Goバージョン（go.modの`go`ディレクティブ）
- 現状: ローカル環境は `go1.26.2`。
- 選択肢:
  - A. ローカル最新に合わせる（`go 1.26`）
  - B. 直近2〜3世代前の枯れたバージョン（例: `go 1.23`）
  - C. `golang.org/x/tools`の対応バージョンに合わせる
- デフォルト案: **B寄りでx/toolsとの整合を取る**。`go vet`のvettoolやanalysisパッケージを使う都合上、CI環境・利用者環境の両方で広く使えることを優先し、最新に固執しない。

### 1.3 リポジトリのディレクトリレイアウト
- 未決定: `cmd/`配下の構成、共通ロジックの置き場所（OQ-1: D1/D2間のコアロジック共有方法）。
- 選択肢:
  - A. `internal/`配下に共通ロジックを集約し、`cmd/importalias`（D1）・`cmd/goimportalias`（D2）は薄いドライバとする
  - B. ルート直下にpackageを置き、cmdは呼び出すだけ
- デフォルト案: **A**。`internal/`によって外部からの誤用を防ぎつつ、D1/D2で確実にロジックを共有できる。

### 1.4 D1バイナリの名前・配置
- 現状: origin.mdのFR-6.2に `go vet -vettool=$(which importalias)` という記述があり、D1バイナリ名は暗黙に `importalias` と読み取れる。FR-7.17でD2は明示的に `goimportalias`（`cmd/goimportalias`）と確定済み。
- デフォルト案: D1は `cmd/importalias/main.go` に配置し、`singlechecker.Main` を呼ぶだけの薄いmainとする。

## 2. 設定ファイルの正確なスキーマ（5.3/5.4節で「実装時に確定してよい」とされている部分）

### 2.1 alias値の型
- 選択肢:
  - A. `string | string[]`（解決済みは文字列、tie未解決は2要素配列、5.4節のイメージ通り）
  - B. 常にオブジェクト `{ "resolved": "..." } | { "candidates": ["...", "..."] }`
- デフォルト案: **A**。origin.md 5.3/5.4節のイメージに忠実で、人間が読んだ時にも直感的。

### 2.2 出力時のフォーマット規則（キー順・インデント）
- 未決定: 設定ファイルを自動生成・更新する際（FR-7.3）に、diffが安定するかどうか。
- デフォルト案: 2スペースインデント、`packages`内のスコープキー・各スコープ内のimport pathキーはアルファベット順にソートして出力。tie候補の2要素配列も昇順ソート。

### 2.3 generated file判定の正規表現
- デフォルト案: 標準的な `^// Code generated .* DO NOT EDIT\.$`（golang.org/s/generatedcode 準拠）をファイル先頭のコメント帯から探索する。

### 2.4 FR-5.7「切り替えられること」の実現方法
- 課題: 設定ファイルのトップレベルは`packages`/`ignore`の2キー固定（FR-5.9）なので、generated file判定のON/OFFを設定ファイルに置けない。
- デフォルト案: CLIフラグ（D2: `-skip-generated`）・analyzerフラグ（D1: `-importalias.skip_generated`）で提供、デフォルト値は`true`（スキップする）。

## 3. 決定ロジックの実装詳細

### 3.1 FR-6.11（同一alias・複数path）のauto-fix方針（OQ-5）
- 選択肢:
  - A. v1では検出のみ、auto-fixは提供しない
  - B. 何らかのヒューリスティック（例: import path文字列長が短い方を優先等）で無理やり決定する
- デフォルト案: **A**。origin.md自身も「保留してよい」としており、無理に決めるとかえって危険な自動書き換えになりうる。

### 3.2 strictオプションの提供経路
- デフォルト案: 設定ファイルには保存せず、実行時オプション（D2: `-strict`、D1: `-importalias.strict`）としてのみ提供する。

## 4. D2 CLIの実装詳細

### 4.1 フラグ一覧
- デフォルト案:
  - `-fix`（bool, default false）
  - `-config string`（default ""、省略時はモジュールルート探索）
  - `-strict`（bool, default false）
  - `-skip-generated`（bool, default true）
  - 対象パッケージは非フラグ引数（`./...`など）で指定

### 4.2 モジュールルート探索方法
- デフォルト案: カレントディレクトリから親ディレクトリを`go.mod`が見つかるまで辿るシンプルな実装。`go/packages`任せにはしない（軽量に保つ）。

### 4.3 FR-7.21の識別子解決手法
- 課題: 「実際のデフォルト識別子名」「既存識別子との衝突有無」を正しく判定するには型情報が必要。
- デフォルト案: `golang.org/x/tools/go/packages` を使い型情報付き（`NeedName|NeedTypes|NeedTypesInfo|NeedSyntax`等）でロードし、`go/types`の情報を用いて意味的に安全な衝突判定を行う。

### 4.4 終了コード
- デフォルト案: `0`=問題なし、`1`=不整合検出（tie未解決含む）、`2`=実行時エラー（I/O・設定ファイル不正等）。

## 5. D1 Analyzerの実装詳細

### 5.1 Analyzer変数の置き場所
- デフォルト案: `internal/analyzer`パッケージに`var Analyzer = &analysis.Analyzer{...}`を定義。`cmd/importalias/main.go`はこれを`singlechecker.Main`に渡すだけ。golangci-lintからの利用（FR-6.3）もこの変数を参照する形で自明に可能になる。

### 5.2 多数決の集計単位（FR-6.17）
- デフォルト案: テスト・非テストを区別せず合算集計（origin.md自身が「実装上容易な方でよい」としている）。

## 6. 依存ライブラリ方針
- デフォルト案: 標準ライブラリ＋`golang.org/x/tools`（analysis系一式、`go/packages`、`analysistest`）のみ。CLIフラグは標準`flag`パッケージ。外部CLIフレームワーク（cobra等）は採用しない（軽量な`go vet`エコシステムの流儀に合わせる）。

## 7. テスト戦略の具体化（12節の実装）
- D1: `analysistest.Run` + `testdata/src/<pkg>/...` + `// want`コメント、という標準パターン。
- D2: golden file方式（`testdata/fix/<case>/input` と `.../golden`を比較）。
- config: (Un)MarshalJSONの往復をtable-driven testで確認。

## 8. CI・配布・ライセンス
- CI: GitHub Actionsで`go build ./...` / `go vet ./...` / `go test ./...`。
- 配布: v1は`go install`のみ、goreleaser等の自動リリースはスコープ外。
- ライセンス: 未決定（デフォルト案としてMITを想定するが要ユーザー確認）。
- 公開先（OQ-2）: 未決定のまま。

## 9. 明示的にスコープ外へ送る論点
- OQ-4（go.work下の「自モジュール」判定）: v1は標準ツールチェーン任せで非対応。
- OQ-6（workspace横断の共有設定ファイル）: 引き続き未検討のまま次フェーズへ。
