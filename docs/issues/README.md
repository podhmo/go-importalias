# 実装 issue 索引

`docs/01decision.md`（実装決定 v1.6）の DEC-10.1 実装着手順の残作業を、独立に着手・テストできる単位に分割した issue 群。運用ルール（ライフサイクル）は [ADR-01](../adr/01-issue-workflow.md)、1 issue を回す実装ループ（検証コマンド）は [ADR-02](../adr/02-issue-impl-loop-commands.md) を参照。

## ライフサイクル

- このディレクトリ直下 = **open**（着手可能・進行中）。
- 終了条件をすべて満たした issue は `closed/` へ**移動**する（内容は履歴として残す）。
- 本索引は **open のみ**を一覧する。closed は [`closed/`](./closed/) を参照。

## 現状（起票時点）

コードは walking skeleton。`go test ./...` は green だが、実機能は **FR-6.10（同一 path → 複数 alias）の検出**と、その **rename ケースの auto-fix** のみ。以下の issue で D1（go vet analyzer）を完成させ、D2（CLI）を段階的に立ち上げ、auto-fix を拡張する。

## open 一覧

### D1（go vet analyzer）完成系

| # | issue | 概要 | 対象 DEC/FR |
|---|---|---|---|

### D2（CLI）基盤

| # | issue | 概要 | 対象 DEC/FR |
|---|---|---|---|
| 06 | [cli-scan-mode](./06-cli-scan-mode.md) | CLI スキャン（module root・go/packages・報告・終了コード） | DEC-4.2〜4.5 / DEC-11.7 / DEC-11.21 |
| 07 | [cli-config-generate](./07-cli-config-generate.md) | CLI の config 生成・更新（`-fix` なし既定動作） | FR-7.2〜7.4 / DEC-11.8 / DEC-11.13 |
| 08 | [cli-fix-apply](./08-cli-fix-apply.md) | CLI `-fix`：auto-fix 適用とファイル書き戻し | FR-7.5〜7.7 / FR-7.18 / DEC-11.7 |

### auto-fix 拡張（astutil）

| # | issue | 概要 | 対象 DEC/FR |
|---|---|---|---|
| 09 | [fr6.11-autofix](./09-fr6.11-autofix.md) | FR-6.11 の auto-fix（import path 追加・削除） | DEC-3.1 / DEC-11.20 / DEC-11.6 |
| 10 | [fr6.16-autofix](./10-fr6.16-autofix.md) | FR-6.16 の auto-fix（重複 import 統合・削除） | FR-7.16 / DEC-11.20 |

### ドキュメント・インフラ

| # | issue | 概要 | 対象 DEC/FR |
|---|---|---|---|
| 11 | [docs-fix-cases](./11-docs-fix-cases.md) | `docs/fix-cases.md` 新設・既存ケース明文化 | DEC-7.2 |
| 12 | [infra-ci-license-readme](./12-infra-ci-license-readme.md) | CI・LICENSE・README | DEC-8.1 / DEC-8.3 / DEC-8.2 |

### バックログ（v1 必須ではない）

| # | issue | 概要 | 対象 DEC/FR |
|---|---|---|---|
| 13 | [dec11.22-collision-precision](./13-dec11.22-collision-precision.md) | 識別子衝突判定の精密化（方針のみ確定・実装は任意） | DEC-11.22 |

## 依存・着手順

```
01 genfile ─┐
02 FR6.11 ──┼─→ 04 analyzer-config ─→ 05 cmd-scaffold ─→ 06 cli-scan ─→ 07 cli-config ─→ 08 cli-fix
03 FR6.16 ──┘                                                                              │
11 fix-cases（05以降と並行、先に土台）───────────────────────────→ 09 FR6.11-fix ─────────┤
                                                                   10 FR6.16-fix ─────────┘
12 infra（いつでも可。05 で bin 化後に CI が意味を持つ）
13 DEC-11.22（バックログ・任意）
```

- **01〜04**: D1 完成（read-only、既存ハーネス上で完結）。相互にほぼ独立で並行着手可能。
- **05〜08**: CLI を段階的に育てる直列パイプライン（骨格 → 読み取り → config 書き → fix 書き）。
- **09/10**: astutil を要する auto-fix 拡張。08 の後、または `internal/fix` 単体テスト上で着手。11（fix-cases）を先に土台化しておく。
- **12**: いつでも着手可。05 でバイナリが build できるようになると CI の `go vet -vettool=` 検証が意味を持つ。
- **13**: v1 スコープ外の任意タスク。
