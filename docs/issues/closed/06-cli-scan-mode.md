# 06: CLI スキャン（module root・go/packages・報告・終了コード）

## 目的

CLI モードの読み取り系を実装する。対象パッケージを `go/packages` で型情報付きにロードし、パッケージ毎に `scan.FromFiles` → `decide.Decide` を回して不整合を人間可読に報告し、終了コードを返す。この issue ではまだファイル書き込み（config 生成・fix）はしない。

## 対象 DEC・FR

- DEC-4.2（module root 探索は `go.mod` を親方向に辿る自前実装。`go env` 等に依存しない）
- DEC-4.3 / DEC-11.21（`go/packages` で型情報付きロード。`scan`/`fix` 双方が使う型情報のロードは CLI の責務。`Scopes` 含む）
- DEC-4.4（終了コード 0/1/2）
- DEC-4.5 / FR-7.15（人間可読なプレーンテキスト出力。「何を検出したか」「どこで tie が保留されたか」）
- DEC-11.7（マルチパッケージは逐次実行）

## 変更ファイル

- `cmd/goimportalias/`（`runCLI` の実装。`flag` で対象パッケージ（非フラグ引数 `./...` 等）を受け、module root 探索、`packages.Load`（`NeedName|NeedTypes|NeedTypesInfo|NeedSyntax|NeedImports|NeedDeps` 相当）、パッケージ毎逐次に scan→decide、報告、終了コード算出）
- 必要なら `cmd/goimportalias/` 内に出力整形・module root 探索の小関数を分ける（副作用整形は cmd のみ＝DEC-11.4）

## 実装メモ

- `packages.Load` の `Config.Mode` に型情報・syntax を含める。`types.Info` に `Scopes` が入るようにする（fix の衝突判定で必須。`docs/02notice.md` 第8回の落とし穴）。
- パッケージ毎に `scan.FromFiles(fset, files, info, scan.Options{Package: pkgPath})` → `decide.Decide(occs, cfg, opts)`。cfg は 07 まで nil でよいが、`-config`/module root の下地はここで用意しておくと 07 が楽。
- 終了コード: 0=不整合なし、1=不整合検出（tie 保留含む）、2=実行時エラー（ロード失敗・I/O 等）。DEC-4.4。
- フラグ本体（`-fix`/`-config`/`-strict`/`-skip-generated`）の定義は DEC-4.1。この issue では最低限 `-strict`/`-skip-generated` をスキャンに反映し、`-fix`/`-config` は宣言だけして後続で使う形でよい。

## 終了条件

- [ ] 既知の不整合を持つ testdata（別モジュール or testmodule）に CLI を exec する統合テストで、(a) 不整合が報告される (b) tie が保留として分かる形で出力される。
- [ ] 終了コード: 不整合ありで 1、なしで 0、ロード失敗（存在しないパッケージ指定等）で 2。
- [ ] module root 探索が `go.mod` を親方向に正しく見つける（サブディレクトリから起動しても解決）。
- [ ] この issue ではファイル書き込みが発生しないことを確認。
- [ ] `go test ./...` 全 green。

## 依存

05（cmd 骨格・モード分岐）が closed であること。
