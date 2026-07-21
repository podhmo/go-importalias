# 実装ドラフト（repomix風・全体俯瞰版）

- **日付**: 2026-07-21
- **目的**: 特定の1パッケージを丁寧に作り込むのではなく、リポジトリ全体（`go.mod`〜`cmd/`〜`internal/*`）をラフに一通りコードとして書いてみることで、「設計原則・コーディング原則としてまだ決まっていないこと」を洗い出す。細かい実装の正しさより、各ピースがどう繋がるはずかという構造の方を優先している。
- **確信度タグの意味**:
  - `[検証済み]` … 実際に `go build`/`go vet`/`go doc` 等で動作・実在を確認したもの。
  - `[未検証]` … 標準的なAPIの使い方として妥当と思われるが、実際に組んで動かしてはいないもの。
  - `[設計上の仮定]` … ここで初めて「こうであるはず」と勝手に置いた設計判断。`docs/01decision.pre.md`の質問の元ネタになっている。
- 一部の関数シグネチャ・型はこのドラフトのためにその場ででっち上げたもの（特に`internal/shape`・`internal/scan`・`internal/decide`・`internal/fix`）であり、正式なAPIではない。
- **第2回からの更新**: PRE-1〜PRE-10がすべて`docs/01decision.md`のDEC-11.1〜DEC-11.10として確定した（`internal/model`→`internal/shape`への改名、D1の`pass.Module`経由config読み込み、`internal/fix`への型情報の明示引数渡し、副作用境界、エラー表現、fix実装方針、逐次実行、差分マージ、コード言語、テスト方針）。このドラフトもそれに合わせて更新し、確定した設計の上でさらに一段深く書いてみて新たな論点が無いか確認する（第3回）。

---

```
================================================================
File: go.mod
================================================================
```

```go
module github.com/podhmo/go-importalias

go 1.26

require golang.org/x/tools v0.48.0
```

`[検証済み]` DEC-1.1/DEC-1.2/DEC-6.1どおり。`go get golang.org/x/tools@latest`が実際に`v0.48.0`を解決できることをこの作業中に確認済み（ネットワーク経由でのモジュール取得が可能な環境だった）。

```
================================================================
File: analyzer.go   (package importalias、モジュールルート)
================================================================
```

```go
package importalias

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/podhmo/go-importalias/internal/decide"
	"github.com/podhmo/go-importalias/internal/scan"
	"github.com/podhmo/go-importalias/internal/shape"
)

var Analyzer = &analysis.Analyzer{
	Name:     "importalias",
	Doc:      "reports inconsistent import path <-> alias mappings within a package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	strictFlag        bool
	skipGeneratedFlag bool
)

func init() {
	Analyzer.Flags.BoolVar(&strictFlag, "strict", false, "always treat differing alias/path pairs as a tie")
	Analyzer.Flags.BoolVar(&skipGeneratedFlag, "skip_generated", true, "skip files with a generated-code marker")
}

func run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	occs := scan.FromInspector(insp, pass.Fset, scan.Options{
		SkipGenerated: skipGeneratedFlag,
	})
	// [設計上の仮定] scan.FromInspectorというAPIをでっち上げた。inspectorはノード種別で
	// フィルタする道具なので、import宣言だけを見るなら普通に pass.Files を
	// ast.Inspect で舐める方が素直かもしれない。inspect.Analyzerへの依存自体が
	// 本当に必要かどうかは未検証。

	// DEC-11.2: pass.Module 経由でモジュールルートを推定し importalias.json を
	// 読み込む。読めなければstderrにログを出し「設定ファイルなし」として続行する。
	//
	// [設計上の仮定・未検証] pass.Module は analysis.Pass の実在フィールドだが、
	// ドキュメント上「possibly nil in some drivers」と明記されている
	// [検証済み: go doc]。go vet -vettool= 経由（unitcheckerドライバ）で
	// 実際に埋まるかどうかは未確認。埋まったとしても pass.Module.Dir が
	// 確実にモジュールルート（go.modのあるディレクトリ）と一致するかも未検証。
	var cfg *shape.File
	if pass.Module == nil {
		fmt.Fprintln(os.Stderr, "importalias: pass.Module is unavailable; proceeding without a config file")
	} else {
		path := filepath.Join(pass.Module.Dir, "importalias.json")
		loaded, err := shape.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "importalias: failed to load %s: %v; proceeding without a config file\n", path, err)
		} else {
			cfg = loaded
		}
	}
	decisions := decide.Decide(occs, cfg, decide.Options{Strict: strictFlag})

	for _, d := range decisions {
		if d.IsTie() {
			continue
			// [設計上の仮定] タイの場合、D1は診断しない設計にしてみた。
			// しかしFR-6.10/FR-6.11は「不整合として検出する」とだけ言っており、
			// 「タイという名の不整合そのもの」を診断すべきかは明記されていない。
			// 診断すべきだとすると go vet の出力に「要:設定ファイルで解消してください」
			// 的なメッセージが出せると親切だが、そのメッセージ文言も未検討。
		}
		for _, bad := range d.Inconsistent {
			pass.Reportf(bad.Pos, "import %q should use alias %q, not %q (package-wide majority)", d.Path, d.WantAlias, bad.Alias)
			// [検証済み] pass.Reportf(pos token.Pos, format string, args ...any) は実在するAPI。
		}
	}
	return nil, nil
}
```

`[未検証]` `analysis.Analyzer`・`inspect.Analyzer`・`inspector.Inspector`・`pass.Reportf`は実在を`go doc`で確認したが、この`run`関数自体を実際にコンパイル・実行してはいない。

```
================================================================
File: cmd/goimportalias/main.go
================================================================
```

```go
package main

import (
	"os"
	"strings"

	"golang.org/x/tools/go/analysis/unitchecker"

	importalias "github.com/podhmo/go-importalias"
)

func main() {
	if looksLikeVetToolInvocation(os.Args) {
		// unitchecker.Main は os.Args を自分で（グローバルなflag.CommandLine経由で）
		// 読みに行く。呼び出す前にこちらでパースし直したりflag.Parse()したり
		// してはいけない（二重パースになる）。
		// [検証済み] unitchecker.go のソースを読み、Main内部で
		// analysisflags.Parse(analyzers, true) → flag.Args() という順で
		// os.Args を直接見ていることを確認した。
		unitchecker.Main(importalias.Analyzer)
		return
	}
	runCLI(os.Args[1:])
}

// looksLikeVetToolInvocation は go vet -vettool= が実際に行う2種類の呼び出し
// （バージョンハンドシェイク "-V=full" と、設定用の "*.cfg" ファイル1個だけを渡す
// 呼び出し）を判定する。
//
// [設計上の仮定] 「*.cfgでさえあれば常にvetツールモード」という単純な判定にしたが、
// ユーザーが誤って "goimportalias somepkg.cfg" のようなCLI引数を渡した場合に
// 誤判定しうる。DEC-1.4は「具体的な判定条件は実装時に go vet -vettool= の実挙動を
// 見ながら固める」としているので、この関数の中身自体がDEC-1.4が要求する
// 「実験で決める」対象そのもの。
func looksLikeVetToolInvocation(args []string) bool {
	if len(args) == 2 && args[1] == "-V=full" {
		return true
	}
	if len(args) == 2 && strings.HasSuffix(args[1], ".cfg") {
		return true
	}
	return false
}

func runCLI(args []string) int {
	// [設計上の仮定] flagセット・-config探索・スキャン・(-fixなら)修正・終了コード算出、
	// という一連の流れをここに書く想定。DEC-4.1〜DEC-4.4で決まっている内容を
	// 呼び出す薄い配線層のつもりだが、実際に書き始めるとどこまでを
	// internal/cli 的な別パッケージに切り出すか、cmd/goimportalias/main.go に
	// 直接書き下すかは決めていない（過去のADRにも記述なし）。
	panic("not implemented in this sketch")
}
```

`[未検証]` `unitchecker.Main`のシグネチャ（`func Main(analyzers ...*analysis.Analyzer)`、戻り値なし＝内部で`os.Exit`する）はソースで確認済み。モード分岐の実際の判定条件は未検証（DEC-1.4自身が「実装時に実挙動を見て固める」としている部分）。

```
================================================================
File: internal/shape/model.go   (DEC-11.1: 共有ドメイン型。config.goと同一パッケージ)
================================================================
```

```go
// package shape は scan → decide → fix / analyzer.go の間でやり取りする
// 共有ドメイン型（本ファイル）と、設定ファイルの型・読み書き（config.go）を
// まとめて置く場所（DEC-11.1で確定。当初案の internal/model は不採用）。
package shape

import "go/token"

// Occurrence は「あるファイルのある位置で、あるimport pathがあるaliasで
// importされていた」という1件の生データ (FR-6.10/FR-6.11の検出対象そのもの)。
type Occurrence struct {
	Path  string // import path (例: "github.com/pkg/errors")
	Alias string // 実際に使われていたalias。無aliasなら宣言パッケージ名と一致する文字列を入れるのか、
	// 空文字にして「無alias」を表すのかは [設計上の仮定・要検討]。
	Pos     token.Pos
	Package string // パッケージパス（決定ロジックの集計単位）
	IsTest  bool   // _test.go 由来かどうか（FR-6.17: 合算するので実質未使用になるかもしれない）
}

// Decision は decide パッケージが1つの (Package, Path) 組について下した判定結果。
type Decision struct {
	Package      string
	Path         string
	WantAlias    string // 多数決 or 設定ファイルにより「正」とされたalias
	Tie          bool   // タイで未解決かどうか
	TieCandidate [2]string
	Inconsistent []Occurrence // WantAliasと矛盾する出現（＝診断・auto-fixの対象）
}

func (d Decision) IsTie() bool { return d.Tie }
```

`[設計上の仮定・要検討]` パッケージの置き場所（`internal/shape`）自体はDEC-11.1で確定済み。残る論点は`Occurrence.Alias`の「無alias」表現（空文字 vs 宣言パッケージ名と一致する文字列）で、これはまだどのDEC-にも明文化されていない。次回の`docs/01decision.pre.md`候補。

```
================================================================
File: internal/scan/scan.go
================================================================
```

```go
package scan

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/ast/inspector"

	"github.com/podhmo/go-importalias/internal/shape"
)

type Options struct {
	SkipGenerated bool
}

// FromInspector は1パッケージ分のASTから import 出現 (FR-6.10/6.11/6.16) を集める。
//
// [設計上の仮定] inspector.Inspector を受け取る形にしたが、import宣言だけを
// 見るのであれば ast.File.Imports（各ファイルのimport一覧はASTが自動で
// 集約してくれる）を直接舐める方が、inspector を要求する分の依存
// （inspect.Analyzerへの Requires）を増やさずに済むかもしれない。
// 「なぜinspectorを使うのか」の理由をまだ言語化できていない。
func FromInspector(insp *inspector.Inspector, fset *token.FileSet, opts Options) []shape.Occurrence {
	var occs []shape.Occurrence
	nodeFilter := []ast.Node{(*ast.File)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		file := n.(*ast.File)
		if opts.SkipGenerated && isGenerated(file) {
			return
		}
		for _, imp := range file.Imports {
			path := mustUnquote(imp.Path.Value)
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			}
			occs = append(occs, shape.Occurrence{
				Path:  path,
				Alias: alias, // [設計上の仮定] 無aliasは空文字。decide側もこの規約を前提にする必要がある。
				Pos:   imp.Pos(),
			})
		}
	})
	return occs
}

func isGenerated(f *ast.File) bool {
	// [設計上の仮定] DEC-2.3の正規表現 `^// Code generated .* DO NOT EDIT\.$` を
	// f.Comments の先頭コメント群に対して適用する想定だが、このスケッチでは
	// 実装を省略した（internal/genfileに切り出す想定なので、ここからはそれを呼ぶだけになるはず）。
	return false
}

func mustUnquote(s string) string {
	// [未検証] strconv.Unquote を使うのが標準的なやり方のはずだが、このスケッチでは省略。
	return s
}
```

`[設計上の仮定]` `Package`フィールド（`shape.Occurrence.Package`）をどこで埋めるかをこの関数の中に書けていない。1パッケージ分のPassを1回のFromInspector呼び出しに対応させるなら、呼び出し側（`analyzer.go`）が後から埋めるのか、`Options`に`PackagePath string`を足すのか、決めていない。

```
================================================================
File: internal/decide/decide.go
================================================================
```

```go
package decide

import (
	"github.com/podhmo/go-importalias/internal/shape"
)

type Options struct {
	Strict bool
}

// Decide は4節の決定ロジック（優先順位→多数決→タイ）を適用する。
// cfg が nil の場合は設定ファイルなしとして扱い、常に多数決のみで決める。
//
// [設計上の仮定] cfg *shape.File を直接受け取る形にしたが、shape.File は
// JSON往復のためのDTOであり、decideが本当に欲しいのは「(package, path) -> alias」
// のような検索しやすい形かもしれない。DTOをそのまま決定ロジックに渡すのか、
// shape側に「Lookup(pkg, path string) (alias string, ok bool)」のような
// 問い合わせ用メソッドを生やすのか（優先順位＝FR-4.1〜4.3の「明示パッケージ名 >
// foo/... > * 」をshape側に持たせるかdecide側に持たせるか）は未確定。
func Decide(occs []shape.Occurrence, cfg *shape.File, opts Options) []shape.Decision {
	// [設計上の仮定・第3回で判明] FR-6.10（同一path→複数alias）とFR-6.11
	// （同一alias→複数path）は、path方向のグルーピングだけでは同時にカバー
	// できない。ここでは2系統の集計を分けて持つ設計にしてみた:
	//   - byPath: (Package, Path) -> []Occurrence  … FR-6.10・4.1〜4.3の主判定
	//   - byAlias: (Package, Alias) -> map[Path]bool … FR-6.11の重複検出専用
	// byAliasの結果をどうDecision型（現状Path単位の構造）に落とし込むかが
	// まだ決まっていない。Decisionにdistinct fieldを足すのか、別のスライスで
	// 返すのか（例えば []AliasCollision のような新型）は未確定のまま。
	byPath := map[string][]shape.Occurrence{} // key = Package + "\x00" + Path
	byAlias := map[string]map[string]bool{}   // key = Package + "\x00" + Alias -> set of Path
	for _, o := range occs {
		pathKey := o.Package + "\x00" + o.Path
		byPath[pathKey] = append(byPath[pathKey], o)

		aliasKey := o.Package + "\x00" + o.Alias
		if byAlias[aliasKey] == nil {
			byAlias[aliasKey] = map[string]bool{}
		}
		byAlias[aliasKey][o.Path] = true
	}

	var out []shape.Decision
	for _, group := range byPath {
		out = append(out, decideOne(group, cfg, opts))
	}
	// [未実装] byAlias の結果（FR-6.11: 1つのaliasに複数のpathが対応している
	// ケース）を out にどう反映するかが、このスケッチではまだ書けていない。
	return out
}

func decideOne(group []shape.Occurrence, cfg *shape.File, opts Options) shape.Decision {
	// [設計上の仮定・大きく未検証] ここに4.1節の優先順位判定 → 4.2節の多数決 →
	// 4.3節のタイ処理、を書く想定。
	panic("not implemented in this sketch")
}
```

`[設計上の仮定]` このファイルは最も雑い。特に「FR-6.10（同一path→複数alias）」と「FR-6.11（同一alias→複数path）」を1つの集計データ構造でまかなえるのか、2系統に分けるべきかが、実際に書いてみて初めて浮かんだ疑問。

```
================================================================
File: internal/shape/config.go   (DEC-11.1: model.goと同じ package shape)
================================================================
```

```go
package shape

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

type File struct {
	Packages map[string]map[string]AliasValue `json:"packages"`
	Ignore   []string                          `json:"ignore,omitempty"`
}

func NewFile() *File {
	return &File{Packages: map[string]map[string]AliasValue{}}
}

type AliasValue struct {
	Resolved string
	Tie      []string
}

func (v AliasValue) IsTie() bool      { return len(v.Tie) > 0 }
func (v AliasValue) IsResolved() bool { return !v.IsTie() }

func (v AliasValue) MarshalJSON() ([]byte, error) {
	if v.IsTie() {
		if len(v.Tie) != 2 {
			return nil, fmt.Errorf("tie candidates must have exactly 2 elements, got %d", len(v.Tie))
		}
		sorted := append([]string(nil), v.Tie...)
		sort.Strings(sorted)
		return json.Marshal(sorted)
	}
	return json.Marshal(v.Resolved)
}

func (v *AliasValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v.Resolved, v.Tie = s, nil
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("alias value must be a string or a 2-element array: %w", err)
	}
	if len(arr) != 2 {
		return fmt.Errorf("tie candidates must have exactly 2 elements, got %d", len(arr))
	}
	v.Resolved, v.Tie = "", arr
	return nil
}

// DEC-11.5: Load/Save don't need a sentinel error of their own — a missing
// file is not an error case (handled below), and every other failure here
// is already the kind of "runtime error" that DEC-4.4's exit code 2 covers
// without needing errors.Is to distinguish it from anything else.

func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	f := NewFile()
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return f, nil
}

func Save(path string, f *File) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// Merge combines a freshly-scanned File into an existing one, per DEC-11.8's
// "preserve merge": package-scope keys present in fresh overwrite the
// corresponding entry in existing; package-scope keys absent from fresh
// (out of this scan's scope) are carried over from existing unchanged.
//
// [設計上の仮定・第3回で新たに判明] DEC-11.8は「スキャン範囲外のpackageキー」
// の扱いは決めたが、「スキャン範囲内のpackageキーの中で、特定のimport pathが
// 今回freshに現れなかった場合」（例えばimport自体が削除された、あるいは
// tieが解消されて設定ファイルへの記載が不要になった等）の扱いまでは決めて
// いない。ここでは「freshのpackageキーはexistingの同キーをまるごと置き換える」
// という単純な方式で実装してみたが、これだと「同じpackage内の他のimport path
// に対する既存のtie候補指定」まで一緒に消えてしまう。これは要確認の新論点。
func Merge(existing, fresh *File) *File {
	out := NewFile()
	for pkg, entries := range existing.Packages {
		out.Packages[pkg] = entries
	}
	for pkg, entries := range fresh.Packages {
		out.Packages[pkg] = entries // package全体を置き換える。上記コメント参照。
	}
	out.Ignore = existing.Ignore
	return out
}
```

`[検証済み]` 型定義・Marshal/UnmarshalJSON・Load/Saveの部分は、前回のドラフトで実際に`go build`/`go vet`が通ることを確認済み（ロジックは同一、`package config`→`package shape`・エラー文字列からの`importalias: `プレフィックス除去のみ変更、DEC-11.5・DEC-11.9反映）。`Merge`はDEC-11.8の方針で実装してみたが、上記コメントの通りpackage内のpath単位の粒度で新たな未決事項が見つかった。

```
================================================================
File: internal/fix/fix.go
================================================================
```

```go
package fix

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/podhmo/go-importalias/internal/shape"
)

// ApplyToFile rewrites import declarations and qualified-identifier
// references (pkg.Symbol) in a single file to match the given decisions
// (FR-7.7). Tied decisions are skipped (FR-7.6).
//
// DEC-11.3: the caller passes typesInfo/typesPkg explicitly; this package
// never calls golang.org/x/tools/go/packages itself.
// DEC-11.6: import add/delete goes through astutil; qualified-identifier
// rewriting is verified against typesInfo.Uses before touching a
// *ast.SelectorExpr, so a same-named local variable is never mistaken for
// the import. Rewriting back to source goes through go/format.Node and
// accepts the resulting gofmt reformatting as a tradeoff.
//
// [検証済み] astutil.AddNamedImport / astutil.DeleteNamedImport は実在するAPI
// （golang.org/x/tools v0.48.0で go doc により確認）。
//
// [設計上の仮定・未検証] SelectorExpr の書き換え判定は
// `typesInfo.Uses[sel.X.(*ast.Ident)] == expectedPkgName`（対象importが
// 束縛する *types.PkgName と一致するか）で行える見込みだが、実際に
// go/packages でロードしたASTに対して動かして確認してはいない。
func ApplyToFile(fset *token.FileSet, file *ast.File, typesInfo *types.Info, typesPkg *types.Package, decisions []shape.Decision) (changed bool, err error) {
	for _, d := range decisions {
		if d.IsTie() {
			continue // FR-7.6: タイ未解決の対象はauto-fix対象外
		}
		for _, occ := range findOccurrencesOf(file, d) {
			for _, sel := range findSelectorExprs(file) {
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					continue
				}
				pkgName, ok := typesInfo.Uses[ident].(*types.PkgName)
				if !ok || pkgName.Imported().Path() != d.Path {
					continue // 別のimportか、importではない識別子を指している
				}
				sel.X = ast.NewIdent(d.WantAlias)
				changed = true
			}
			_ = occ
		}
		// [設計上の仮定] astutil.DeleteNamedImport(fset, file, oldAlias, d.Path) →
		// astutil.AddNamedImport(fset, file, d.WantAlias, d.Path) という
		// 2段階になりそうだが、alias無し(空文字)の場合の
		// AddNamedImport(fset, file, "", path) の扱い（"" を渡すと
		// 無alias importとして追加されるのか？）は未検証。
		_ = astutil.AddImport
	}
	if !changed {
		return false, nil
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return false, err
	}
	return true, nil
}

func findOccurrencesOf(file *ast.File, d shape.Decision) []shape.Occurrence { return d.Inconsistent }
func findSelectorExprs(file *ast.File) []*ast.SelectorExpr {
	var out []*ast.SelectorExpr
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			out = append(out, sel)
		}
		return true
	})
	return out
}
```

`[設計上の仮定・未検証]` DEC-11.3/DEC-11.6を反映し、型情報を明示引数として受け取る形・`types.Info.Uses`による安全確認・`go/format.Node`での書き戻しまで一通り書けたが、実際にサンプルファイルへ通して動作確認はしていない。特に`findOccurrencesOf`と`findSelectorExprs`の組み合わせ方（本来は同一importに属する複数のSelectorExprだけを対象に絞り込むべきで、このスケッチでは全SelectorExprを毎回舐めていて非効率かつ他importの識別子も誤って書き換えかねない）はさらに整理が要る、実装時の詰めどころ。

```
================================================================
File: internal/genfile/genfile.go
================================================================
```

```go
package genfile

import (
	"bufio"
	"bytes"
	"regexp"
)

// DEC-2.3の正規表現。
var generatedPattern = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// IsGenerated はファイル先頭のコメント帯を走査し、generated file
// マーカーを検出したら true を返す。
//
// [未検証] 「ファイル先頭の、最初の非コメント・非空行が現れるまでの範囲」
// (DEC-2.3)を素朴に行単位でスキャンする実装。go/ast の Comments を
// 使わずbyte列を直接舐める形にしたが、この単純さで十分か
// （例えばbuild constraint行 `//go:build ...` との共存など）は未検証。
func IsGenerated(src []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if len(line) >= 2 && line[:2] == "//" {
			if generatedPattern.MatchString(line) {
				return true
			}
			continue
		}
		break
	}
	return false
}
```

`[未検証]` ロジックとしては素直だが、実際にサンプルファイルを通していない。

---

## 第2回時点での雑感（そのまま記録として残す）

- `internal/config`（DEC-2.1由来）だけは前回すでに実際にビルド・vet済みで、確信度が高い。
- それ以外（`scan`/`decide`/`fix`/`model`/`analyzer.go`/`cmd`）は、**「型・関数を書き始めて初めて、パッケージ同士がどういうデータをどう受け渡すべきかが決まっていないことに気づいた」**という発見の方が、個々のAPIの正しさより大きな収穫だった。
- 特に3点、原則レベルで確認したい:
  1. `internal/model`のような共有ドメイン型パッケージを新設してよいか（1.3節のレイアウトにない箱を1つ増やすことになる）。
  2. D1（go vet analyzer）はそもそも設定ファイル（`importalias.json`）を読むのか、読まないのか（FR-4.1の「常に優先」との整合性）。
  3. `internal/fix`が識別子の衝突チェック（FR-7.21）を行うために型情報（`go/types`）をどこから受け取るのか（`internal/scan`や`analyzer.go`とデータフローが繋がっていない）。

これらを `docs/01decision.pre.md` に質問として整理し、PRE-1〜PRE-10としてユーザーに確認、すべて`docs/01decision.md`のDEC-11.1〜DEC-11.10として確定した。

## 第3回：確定した原則の上に書き直して見つかった新たな論点

DEC-11.1〜DEC-11.10を反映して本ドラフトを書き直す中で、次の**新たな**設計上の未決事項が見つかった（前回の3点とは異なり、いずれも「原則は決まったが、その適用の細部でまだ揺れがある」種類の論点）。

1. **`Occurrence.Alias`の「無alias」表現**（`internal/shape/model.go`）: 空文字で表すのか、宣言パッケージ名と一致する文字列を入れるのか、まだどのDEC-にも明文化されていない。`scan`と`decide`の両方がこの規約を前提にするため、地味だが揺れると全体に波及する。
2. **FR-6.10とFR-6.11の集計結果の型**（`internal/decide/decide.go`）: path方向（`byPath`）とalias方向（`byAlias`）の2系統集計が要ることは第2回で判明済みだったが、実際に書いてみると「`byAlias`の検出結果をどう`shape.Decision`に反映するか」（`Decision`に別フィールドを足すか、`[]AliasCollision`のような別型で返すか）がまだ決まっていない。
3. **`shape.Merge`のpackage内の粒度**（`internal/shape/config.go`）: DEC-11.8は「スキャン範囲外のpackageキー」の扱いを決めたが、「スキャン範囲**内**のpackageキーの中で、特定のimport pathが今回freshに現れなかった場合」（importが削除された、tieが解消された等）に既存のtie候補指定ごと消してよいかは未決。
4. **`internal/fix`のSelectorExpr走査範囲**（`internal/fix/fix.go`）: `types.Info.Uses`で安全確認する設計自体はDEC-11.6で確定したが、実際に書くと「ファイル全体のSelectorExprを毎回舐める」素朴な実装は非効率かつ書き換え漏れ・誤爆のリスクがある。対象importに紐づくSelectorExprだけを事前に絞り込む設計が必要そうだが未設計。

これらは前回までの「パッケージ間のデータフロー」レベルの大きな穴とは異なり、より細かい粒度の仕様確定待ち事項のため、次回`docs/01decision.pre.md`に**PRE-11〜PRE-14**として追記するか、あるいは実装しながら`docs/fix-cases.md`的な形で都度確定させる（DEC-7.5と同じ「手を動かしながら決める」方針の対象にする）か、方針そのものをユーザーに確認したい。
