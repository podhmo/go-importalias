# 未確定事項（Pre-Decisions）— 質問リスト

- **日付**: 2026-07-21
- **本書の位置づけ**: `docs/01decision.md`（確定済みの実装決定）を補うための、未確定事項の一時的な置き場。
- **各質問の「推奨（デフォルト）」は実装者側からの提案であり、ユーザーの確認が済むまでは確定事項（DEC-）にはしない。**
- **運用**: 回答・確認が済んだ項目は本書から削除し、`docs/01decision.md`に`DEC-`として追記する。

---

## PRE-23: DEC-11.22 識別子衝突判定の精密化で扱う範囲

**質問**: DEC-11.22 の具体実装では、識別子衝突判定をどこまで精密化するか？

**推奨（デフォルト）**: 「valid input を invalid output にしない」ことを基準に、次の範囲までを実装対象にする。

- rename 後の import name が同一ファイルの別 import name、または同一 package の package-level 宣言と衝突する場合は skip する。
- 使用箇所の位置を考慮した scope lookup を使い、ローカル変数・parameter・named result・type parameter・receiver・`for`/`if`/`switch` init 変数・`range` 変数など、通常の lexical scope で実際に見える同名 object がある場合だけ skip する。
- rename 後の名前が predeclared identifier（`len` など）で、既存 builtin 使用を壊す場合は skip する。既存 builtin 使用がない場合は rewrite 可とする。
- 1 import spec の全使用箇所が安全な場合だけ rewrite し、部分 rewrite・ローカル識別子側 rename・invalid 入力の救済は行わない。

**理由**: `docs/02notice.md`第13回のケース棚卸しでは、上記の範囲なら `shape.NameVisibleAt` の現在の保守的近似を精密化しても、Go として invalid な出力や意味破壊を避けられると判断できた。一方、部分 rewrite やローカル識別子側 rename は変換の責務が大きく変わり、DEC-11.22 の「衝突判定の精密化」を超えるため、別論点に分ける方が安全。

**確認後の反映先**: 推奨どおりなら `docs/01decision.md` の DEC-11.22 または `docs/issues/13-dec11.22-collision-precision.md` に具体実装範囲として追記する。推奨と異なる判断になった場合は、その差分と理由を DEC 側に記録する。
