// Comando mcp-impact: analiza el impacto de un cambio sobre las tools MCP.
//
// Fuente de verdad: docs/mcp/tools/impact.json.
// Uso:
//
//	go run ./scripts/mcp-impact --check-map
//	go run ./scripts/mcp-impact --files internal/services/bill_bulk.go
//	go run ./scripts/mcp-impact --staged --gate
//
// El comando es de desarrollo: no se ejecuta en el iHost ni en el server.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	manifestRelPath = "docs/mcp/tools/impact.json"
	impactHeading   = "## Impacto en tools MCP"
)

var (
	toolNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	endpointRE = regexp.MustCompile(`^(GET|POST|PUT|PATCH|DELETE) /`)
	specFileRE = regexp.MustCompile(`^docs/specs/SPEC-[0-9]{3}-[^/]+\.md$`)
	branchRE   = regexp.MustCompile(`^feature/(SPEC-[0-9]{3})$`)
	specIDRE   = regexp.MustCompile(`^SPEC-[0-9]{3}$`)
	subagentRE = regexp.MustCompile(`(?mi)^\s*(?:[-*]\s*)?(?:\*\*)?Validación subagente(?:\*\*)?\s*:\s*(.+)$`)
)

// Manifest es la fuente de verdad del impacto de cada tool.
type Manifest struct {
	Version     int    `json:"version"`
	Description string `json:"description,omitempty"`
	Tools       []Tool `json:"tools"`
	Shared      Shared `json:"shared"`
}

// Tool declara las dependencias de una tool MCP.
type Tool struct {
	Name         string   `json:"name"`
	Docs         []string `json:"docs"`
	Entrypoints  []string `json:"entrypoints"`
	Dependencies []string `json:"dependencies"`
	Endpoints    []string `json:"endpoints"`
	Tests        []string `json:"tests"`
}

// Shared declara rutas que afectan a todas las tools.
type Shared struct {
	AffectsAll []string `json:"affects_all"`
	Notes      string   `json:"notes,omitempty"`
}

// ToolImpact es el resultado del análisis para una tool.
type ToolImpact struct {
	Name    string   `json:"name"`
	Reasons []string `json:"reasons"`
}

// Report es la salida JSON del analizador.
type Report struct {
	Affected []ToolImpact `json:"affected"`
	Spec     string       `json:"spec,omitempty"`
	OK       bool         `json:"ok"`
	Error    string       `json:"error,omitempty"`
}

// stringList permite repetir --files o pasar una lista separada por comas.
type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			*s = append(*s, item)
		}
	}
	return nil
}

func main() {
	var (
		staged   = flag.Bool("staged", false, "usa los archivos staged (git diff --cached)")
		checkMap = flag.Bool("check-map", false, "valida el manifiesto contra las tools registradas")
		gate     = flag.Bool("gate", false, "exige spec con impacto, valida specs nuevas y corre tests MCP + contrato backend")
		jsonOut  = flag.Bool("json", false, "salida JSON")
		rootFlag = flag.String("root", ".", "raíz del repositorio")
		specFlag = flag.String("spec", "", "spec a validar (ruta o SPEC-XXX)")
		files    stringList
	)
	flag.Var(&files, "files", "archivo o lista separada por comas (repetible)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Uso: %s [--check-map] [--staged] [--gate] [--json] [--spec SPEC-XXX] [archivos...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		exitErr(fmt.Errorf("resolver raíz: %w", err))
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		exitErr(fmt.Errorf("raíz inválida: %s", root))
	}

	manifest, err := loadManifest(filepath.Join(root, manifestRelPath))
	if err != nil {
		exitErr(err)
	}
	if err := validateManifest(root, manifest); err != nil {
		exitErr(err)
	}

	if *checkMap {
		fmt.Printf("[mcp-impact] ✓ manifiesto válido: %d tools registradas y todas sus rutas existen\n", len(manifest.Tools))
		return
	}

	branch := currentBranch(root)
	if *gate && (isMergeInProgress(root) || branch == "main") {
		fmt.Println("[mcp-impact] rama main o merge en curso: se omite el gate de spec (release/bump)")
		return
	}

	changed := append([]string(files), flag.Args()...)
	if *staged {
		changed, err = stagedFiles(root)
		if err != nil {
			exitErr(err)
		}
		if *gate {
			added, err := stagedAddedFiles(root)
			if err != nil {
				exitErr(err)
			}
			if err := validateNewSpecs(root, added); err != nil {
				exitErr(err)
			}
		}
	}
	changed = normalizeFiles(root, changed)

	impact := computeImpact(manifest, changed)

	report := Report{Affected: impact, OK: true}
	if len(impact) == 0 {
		if *jsonOut {
			writeJSON(report)
		} else {
			fmt.Println("[mcp-impact] sin tools afectadas")
		}
		return
	}

	if *jsonOut {
		writeJSON(report)
	} else {
		fmt.Printf("[mcp-impact] tools afectadas: %s\n", strings.Join(toolNames(impact), ", "))
		for _, ti := range impact {
			for _, reason := range ti.Reasons {
				fmt.Printf("  - %s: %s\n", ti.Name, reason)
			}
		}
	}

	if *gate {
		spec, err := findSpec(root, branch, changed, *specFlag)
		if err == nil {
			report.Spec = relPath(root, spec)
		}
		if err != nil {
			report.OK = false
			report.Error = err.Error()
			if *jsonOut {
				writeJSON(report)
			}
			exitErr(err)
		}
		if err := applyGate(root, spec, impact); err != nil {
			report.OK = false
			report.Error = err.Error()
			if *jsonOut {
				writeJSON(report)
			}
			exitErr(err)
		}
	}
}

func loadManifest(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("abrir manifiesto %s: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("leer manifiesto %s: %w", path, err)
	}
	if manifest.Version != 1 {
		return nil, fmt.Errorf("versión de manifiesto no soportada: %d", manifest.Version)
	}
	return &manifest, nil
}

func validateManifest(root string, manifest *Manifest) error {
	if len(manifest.Tools) == 0 {
		return errors.New("el manifiesto no declara tools")
	}
	if len(manifest.Shared.AffectsAll) == 0 {
		return errors.New("el manifiesto no declara rutas compartidas en shared.affects_all")
	}

	seen := make(map[string]bool, len(manifest.Tools))
	for _, tool := range manifest.Tools {
		if !toolNameRE.MatchString(tool.Name) {
			return fmt.Errorf("nombre de tool inválido: %q", tool.Name)
		}
		if seen[tool.Name] {
			return fmt.Errorf("tool duplicada en el manifiesto: %s", tool.Name)
		}
		seen[tool.Name] = true

		if len(tool.Docs) == 0 {
			return fmt.Errorf("%s: falta docs", tool.Name)
		}
		for _, path := range tool.Docs {
			if err := checkFile(root, path); err != nil {
				return fmt.Errorf("%s: docs: %w", tool.Name, err)
			}
		}
		if len(tool.Entrypoints) == 0 {
			return fmt.Errorf("%s: falta entrypoints", tool.Name)
		}
		for _, path := range tool.Entrypoints {
			if err := checkFile(root, path); err != nil {
				return fmt.Errorf("%s: entrypoint: %w", tool.Name, err)
			}
		}
		if len(tool.Dependencies) == 0 {
			return fmt.Errorf("%s: falta dependencies", tool.Name)
		}
		for _, pattern := range tool.Dependencies {
			if err := checkPattern(root, pattern); err != nil {
				return fmt.Errorf("%s: dependency: %w", tool.Name, err)
			}
		}
		if len(tool.Endpoints) == 0 {
			return fmt.Errorf("%s: falta endpoints", tool.Name)
		}
		for _, endpoint := range tool.Endpoints {
			if !endpointRE.MatchString(endpoint) {
				return fmt.Errorf("%s: endpoint inválido %q (formato: METHOD /ruta)", tool.Name, endpoint)
			}
		}
		if len(tool.Tests) == 0 {
			return fmt.Errorf("%s: falta tests", tool.Name)
		}
		for _, path := range tool.Tests {
			if err := checkFile(root, path); err != nil {
				return fmt.Errorf("%s: test: %w", tool.Name, err)
			}
		}
	}

	for _, pattern := range manifest.Shared.AffectsAll {
		if err := checkPattern(root, pattern); err != nil {
			return fmt.Errorf("shared.affects_all: %w", err)
		}
	}

	registered, err := registeredTools(root)
	if err != nil {
		return err
	}
	for name, file := range registered {
		if !seen[name] {
			return fmt.Errorf("la tool %q registrada en %s no está en %s", name, file, manifestRelPath)
		}
	}
	for name := range seen {
		if _, ok := registered[name]; !ok {
			return fmt.Errorf("el manifiesto declara %q pero el código no la registra con mcp.AddTool", name)
		}
	}
	return nil
}

func checkFile(root, path string) error {
	if strings.ContainsAny(path, "*?[") {
		return fmt.Errorf("se esperaba un archivo exacto, no un patrón: %s", path)
	}
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		return fmt.Errorf("no existe %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("%s es un directorio, se esperaba un archivo", path)
	}
	return nil
}

func checkPattern(root, pattern string) error {
	pattern = filepath.ToSlash(pattern)
	if strings.ContainsAny(pattern, "*?[") {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return fmt.Errorf("patrón inválido %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("el patrón %s no coincide con ningún archivo", pattern)
		}
		return nil
	}
	info, err := os.Stat(filepath.Join(root, pattern))
	if err != nil {
		return fmt.Errorf("no existe %s", pattern)
	}
	if strings.HasSuffix(pattern, "/") && !info.IsDir() {
		return fmt.Errorf("%s no es un directorio", pattern)
	}
	return nil
}

func registeredTools(root string) (map[string]string, error) {
	toolsDir := filepath.Join(root, "internal/mcp/tools")
	tools := make(map[string]string)
	err := filepath.WalkDir(toolsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parsear %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !isAddToolCall(call) || len(call.Args) < 2 {
				return true
			}
			literal := toolLiteral(call.Args[1])
			if literal == nil {
				return true
			}
			for _, element := range literal.Elts {
				keyValue, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := keyValue.Key.(*ast.Ident)
				if !ok || key.Name != "Name" {
					continue
				}
				value, ok := keyValue.Value.(*ast.BasicLit)
				if !ok || value.Kind != token.STRING {
					continue
				}
				name, err := strconv.Unquote(value.Value)
				if err == nil {
					rel, _ := filepath.Rel(root, path)
					tools[name] = filepath.ToSlash(rel)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("leer tools registradas: %w", err)
	}
	return tools, nil
}

func isAddToolCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "AddTool" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == "mcp"
}

func toolLiteral(expr ast.Expr) *ast.CompositeLit {
	switch value := expr.(type) {
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			literal, _ := value.X.(*ast.CompositeLit)
			return literal
		}
	case *ast.CompositeLit:
		return value
	}
	return nil
}

func stagedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}
	return splitLines(string(output)), nil
}

func stagedAddedFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "diff", "--cached", "--name-only", "--diff-filter=A")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --cached --diff-filter=A: %w", err)
	}
	return splitLines(string(output)), nil
}

func isMergeInProgress(root string) bool {
	cmd := exec.Command("git", "-C", root, "rev-parse", "-q", "--verify", "MERGE_HEAD")
	return cmd.Run() == nil
}

// validateNewSpecs exige impacto y validación subagente en toda spec nueva,
// aunque el cambio no toque directamente una tool MCP.
func validateNewSpecs(root string, files []string) error {
	validated := 0
	for _, file := range files {
		if !specFileRE.MatchString(file) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			return fmt.Errorf("leer spec nueva %s: %w", file, err)
		}
		section := impactSection(string(content))
		if section == "" {
			return fmt.Errorf("la spec nueva %s no contiene la sección %q", file, impactHeading)
		}
		if value, ok := subagentValidation(section); !ok {
			return fmt.Errorf("la spec nueva %s no registra una Validación subagente válida (valor actual: %q)", file, value)
		}
		validated++
	}
	if validated > 0 {
		fmt.Printf("[mcp-impact] ✓ %d spec(s) nueva(s) con impacto y validación subagente\n", validated)
	}
	return nil
}

func normalizeFiles(root string, files []string) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if filepath.IsAbs(file) {
			if rel, err := filepath.Rel(root, file); err == nil {
				file = rel
			}
		}
		result = append(result, filepath.ToSlash(filepath.Clean(file)))
	}
	return result
}

func computeImpact(manifest *Manifest, files []string) []ToolImpact {
	reasons := make(map[string]map[string]bool, len(manifest.Tools))
	add := func(name, reason string) {
		if reasons[name] == nil {
			reasons[name] = make(map[string]bool)
		}
		reasons[name][reason] = true
	}

	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		for _, pattern := range manifest.Shared.AffectsAll {
			if matchPattern(pattern, file) {
				for _, tool := range manifest.Tools {
					add(tool.Name, fmt.Sprintf("ruta compartida %s (cambio en %s)", pattern, file))
				}
			}
		}
		for _, tool := range manifest.Tools {
			for _, pattern := range tool.Entrypoints {
				if matchPattern(pattern, file) {
					add(tool.Name, fmt.Sprintf("entrypoint %s (cambio en %s)", pattern, file))
				}
			}
			for _, pattern := range tool.Dependencies {
				if matchPattern(pattern, file) {
					add(tool.Name, fmt.Sprintf("dependencia %s (cambio en %s)", pattern, file))
				}
			}
		}
	}

	result := make([]ToolImpact, 0, len(reasons))
	for name, set := range reasons {
		list := make([]string, 0, len(set))
		for reason := range set {
			list = append(list, reason)
		}
		sort.Strings(list)
		result = append(result, ToolImpact{Name: name, Reasons: list})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func matchPattern(pattern, file string) bool {
	pattern = filepath.ToSlash(pattern)
	file = filepath.ToSlash(file)
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(file, pattern)
	}
	if strings.ContainsAny(pattern, "*?[") {
		ok, err := filepath.Match(pattern, file)
		return err == nil && ok
	}
	return pattern == file
}

func currentBranch(root string) string {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func findSpec(root, branch string, changed []string, explicit string) (string, error) {
	if explicit != "" {
		if specIDRE.MatchString(explicit) {
			matches, _ := filepath.Glob(filepath.Join(root, "docs/specs", explicit+"-*.md"))
			if len(matches) == 1 {
				return matches[0], nil
			}
			return "", fmt.Errorf("no se encontró una única spec para %s", explicit)
		}
		path := explicit
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("la spec indicada no existe: %s", explicit)
		}
		return path, nil
	}

	var stagedSpecs []string
	for _, file := range changed {
		if specFileRE.MatchString(file) {
			stagedSpecs = append(stagedSpecs, file)
		}
	}

	branchID := ""
	if match := branchRE.FindStringSubmatch(branch); len(match) == 2 {
		branchID = match[1]
	}
	if branchID != "" {
		matches, _ := filepath.Glob(filepath.Join(root, "docs/specs", branchID+"-*.md"))
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			for _, staged := range stagedSpecs {
				if strings.Contains(staged, branchID) {
					return filepath.Join(root, staged), nil
				}
			}
			return "", fmt.Errorf("hay varias specs para %s: %v", branchID, matches)
		}
	}

	if len(stagedSpecs) > 0 {
		sort.Strings(stagedSpecs)
		return filepath.Join(root, stagedSpecs[len(stagedSpecs)-1]), nil
	}
	return "", fmt.Errorf("no se encontró la spec en curso; creá docs/specs/SPEC-XXX-<slug>.md o usá --spec")
}

func applyGate(root, spec string, impact []ToolImpact) error {
	content, err := os.ReadFile(spec)
	if err != nil {
		return fmt.Errorf("leer spec %s: %w", spec, err)
	}
	section := impactSection(string(content))
	if section == "" {
		return fmt.Errorf("la spec %s no contiene la sección %q", relPath(root, spec), impactHeading)
	}

	var missing []string
	for _, tool := range impact {
		if !containsWord(section, tool.Name) {
			missing = append(missing, tool.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("la spec %s no declara el impacto de: %s", relPath(root, spec), strings.Join(missing, ", "))
	}

	if value, ok := subagentValidation(section); !ok {
		return fmt.Errorf("la spec %s no registra una Validación subagente válida (valor actual: %q)", relPath(root, spec), value)
	}

	cmd := exec.Command("go", "test", "./internal/mcp/...")
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test ./internal/mcp/... falló: %w", err)
	}

	backend := exec.Command("go", "test", "./internal/api", "-run", "TestMCPBackendContract")
	backend.Dir = root
	backend.Stdout = os.Stdout
	backend.Stderr = os.Stderr
	if err := backend.Run(); err != nil {
		return fmt.Errorf("el contrato backend↔MCP falló (go test ./internal/api -run TestMCPBackendContract): %w", err)
	}
	fmt.Printf("[mcp-impact] ✓ %s declara impacto y validación subagente; tests MCP y contrato backend verdes\n", relPath(root, spec))
	return nil
}

func impactSection(content string) string {
	lines := strings.Split(content, "\n")
	start := -1
	for index, line := range lines {
		normalized := strings.TrimSpace(line)
		if strings.HasPrefix(normalized, "## ") && strings.Contains(strings.ToLower(normalized), "impacto en tools mcp") {
			start = index
			break
		}
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), "## ") {
			end = index
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

func subagentValidation(section string) (string, bool) {
	match := subagentRE.FindStringSubmatch(section)
	if len(match) < 2 {
		return "", false
	}
	value := strings.TrimSpace(match[1])
	lower := strings.ToLower(value)
	if value == "" || strings.HasPrefix(lower, "pendiente") || lower == "no" || lower == "n/a" || lower == "-" || lower == "..." {
		return value, false
	}
	return value, true
}

func containsWord(text, word string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(text)
}

func toolNames(impact []ToolImpact) []string {
	names := make([]string, 0, len(impact))
	for _, tool := range impact {
		names = append(names, tool.Name)
	}
	return names
}

func splitLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func writeJSON(report Report) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "[mcp-impact] ⛔ %s\n", err)
	os.Exit(1)
}
