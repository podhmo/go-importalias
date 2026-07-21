# 12: CI・LICENSE・README

## 目的

公開リポジトリ（`github.com/podhmo/go-importalias`）としての最低限のインフラを整える。CI ワークフロー、MIT ライセンス、利用方法を書いた README を用意する。

## 対象 DEC・FR

- DEC-8.1（GitHub Actions で `go build ./...` / `go vet ./...` / `go test ./...`）
- DEC-8.3（ライセンスは MIT）
- DEC-8.2（v1 の配布は `go install .../cmd/goimportalias@latest` のみ。goreleaser 等は v1 スコープ外）
- DEC-8.4（`github.com/podhmo/go-importalias` として公開前提）
- FR-6.2 / FR-7.17（利用経路の主軸は `go vet -vettool=`、CLI 名は `goimportalias`）

## 変更ファイル

- `.github/workflows/ci.yml`（新規。push / PR で 3 コマンドを実行。Go バージョンは go.mod の `go 1.26` に合わせる）
- `LICENSE`（新規。MIT。著作者表記は podhmo）
- `README.md`（新規）

## 実装メモ

- README に含める最低限:
  - プロジェクトの目的（パッケージ内の import path ↔ alias 一貫性の検出・自動修正）。
  - `go vet -vettool=$(which goimportalias) ./...` による利用（D1）。
  - `go install github.com/podhmo/go-importalias/cmd/goimportalias@latest`。
  - CLI（D2）のフラグ一覧（`-fix` / `-config` / `-strict` / `-skip-generated`。DEC-4.1）と終了コード（DEC-4.4）。
  - `importalias.json` の例（`packages` / `ignore`、tie の配列表現。origin 5 節・DEC-2.1）。
  - golangci-lint 連携は Analyzer export により自明である旨（副次的位置づけ、ADR-6）。
- CI の Go バージョンは `1.26`。ローカル環境に合わせた最新前提（DEC-1.2）。利用者環境が古い Go の可能性は許容。

## 終了条件

- [ ] `.github/workflows/ci.yml` が push / PR で `go build ./...` / `go vet ./...` / `go test ./...` を実行し green。
- [ ] `LICENSE` が MIT。
- [ ] `README.md` に vet 利用・`go install`・CLI フラグ一覧・終了コード・`importalias.json` 例が記載されている。

## 依存

CI の `go vet -vettool=` 検証を意味あるものにするには 05（bin 化）以降が望ましいが、3 コマンド CI 自体はいつでも着手可能。README の CLI フラグ節は 05〜08 の確定後に書くと正確。
