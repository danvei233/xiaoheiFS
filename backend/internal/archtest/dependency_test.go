package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDomainDependencyBoundary(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "domain")
	violations := collectImportViolations(t, target, []string{
		"xiaoheiplay/internal/adapter",
		"github.com/gin-gonic/gin",
		"gorm.io/",
		"net/http",
	})
	if len(violations) > 0 {
		t.Fatalf("domain dependency violation(s):\n%s", strings.Join(violations, "\n"))
	}
}

func TestAppDependencyBoundary(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "app")
	violations := collectImportViolations(t, target, []string{
		"xiaoheiplay/internal/adapter",
		"github.com/gin-gonic/gin",
		"gorm.io/",
		"net/http",
	})
	if len(violations) > 0 {
		t.Fatalf("app dependency violation(s):\n%s", strings.Join(violations, "\n"))
	}
}

func TestHTTPAdapterDependencyBoundary(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "adapter", "http")
	violations := collectImportViolations(t, target, []string{
		"xiaoheiplay/internal/adapter/repo/core",
		"xiaoheiplay/internal/adapter/email",
		"xiaoheiplay/internal/adapter/robot",
		"xiaoheiplay/internal/adapter/push",
	})
	violations = dropViolationsWithPrefix(violations, "install.go:")
	if len(violations) > 0 {
		t.Fatalf("http adapter dependency violation(s):\n%s", strings.Join(violations, "\n"))
	}
}

func TestHTTPHandlerNoSetterInjection(t *testing.T) {
	root := projectRoot(t)
	filePath := filepath.Join(root, "internal", "adapter", "http", "handlers.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse handlers.go: %v", err)
	}
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := star.X.(*ast.Ident)
		if !ok || ident.Name != "Handler" {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "Set") {
			violations = append(violations, fn.Name.Name)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("handler setter injection is forbidden: %s", strings.Join(violations, ", "))
	}
}

func TestHTTPHandlerNoConcretePluginManagerField(t *testing.T) {
	root := projectRoot(t)
	filePath := filepath.Join(root, "internal", "adapter", "http", "handlers.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse handlers.go: %v", err)
	}
	targets := map[string]bool{"HandlerDeps": true, "Handler": true}
	var violations []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !targets[ts.Name.Name] {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range st.Fields.List {
				star, ok := field.Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				sel, ok := star.X.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					continue
				}
				if pkg.Name == "plugins" && sel.Sel.Name == "Manager" {
					name := "<anonymous>"
					if len(field.Names) > 0 {
						name = field.Names[0].Name
					}
					violations = append(violations, ts.Name.Name+"."+name)
				}
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("concrete plugins.Manager field in http handler is forbidden: %s", strings.Join(violations, ", "))
	}
}

func TestHTTPProductionNoDirectPluginV1Import(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "adapter", "http")
	violations := collectImportViolations(t, target, []string{"xiaoheiplay/plugin/v1"})
	if len(violations) > 0 {
		t.Fatalf("http production direct plugin/v1 import is forbidden:\n%s", strings.Join(violations, "\n"))
	}
}

func TestSMSEntryHandlersNoPluginManagerUsage(t *testing.T) {
	root := projectRoot(t)
	files := []string{
		filepath.Join(root, "internal", "adapter", "http", "handlers_admin_messaging.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_site_auth.go"),
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(b), "h.pluginMgr") {
			t.Fatalf("sms entry handler must not use h.pluginMgr directly: %s", filepath.Base(path))
		}
	}
}

func TestAutomationSettingsHandlerNoPluginManagerUsage(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers_admin_settings_automation.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(b), "h.pluginMgr") {
		t.Fatalf("automation settings handler must not use h.pluginMgr directly: %s", filepath.Base(path))
	}
}

func TestOrderDetailHandlersNoDirectOrderRepos(t *testing.T) {
	root := projectRoot(t)
	files := []string{
		filepath.Join(root, "internal", "adapter", "http", "handlers_admin_orders_tickets.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_site_orders_wallet.go"),
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(b)
		if strings.Contains(text, "h.orderRepo") || strings.Contains(text, "h.orderItems") || strings.Contains(text, "h.payments.") {
			t.Fatalf("order detail handlers must not use direct order repositories: %s", filepath.Base(path))
		}
	}
}

func TestAuthSecurityHandlerNoDirectUsersRepoUsage(t *testing.T) {
	root := projectRoot(t)
	files := []string{
		filepath.Join(root, "internal", "adapter", "http", "handlers_auth_security.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_site_auth.go"),
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(b), "h.users.") {
			t.Fatalf("auth handlers must not use h.users directly: %s", filepath.Base(path))
		}
	}
}

func TestSiteVPSTicketHandlerNoDirectVPSRepoUsage(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers_site_vps_ticket.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(b), "h.vpsRepo.") {
		t.Fatalf("site vps ticket handler must not use h.vpsRepo directly: %s", filepath.Base(path))
	}
}

func TestSelectedHandlersNoDirectSettingsRepoUsage(t *testing.T) {
	root := projectRoot(t)
	files := []string{
		filepath.Join(root, "internal", "adapter", "http", "handlers_site_auth.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_site_vps.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_probe.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_admin_messaging.go"),
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(b), "h.settings.") {
			t.Fatalf("selected handlers must not use h.settings directly: %s", filepath.Base(path))
		}
	}
}

func TestAdminGoodsUploadsPermissionsHandlerNoDirectRepos(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers_admin_goods_upload_permissions.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(b)
	if strings.Contains(text, "h.uploads.") || strings.Contains(text, "h.permissions.") {
		t.Fatalf("admin goods/uploads/permissions handler must not use h.uploads/h.permissions directly: %s", filepath.Base(path))
	}
}

func TestAdminPluginsHandlerNoDirectPluginManagerOrRepoUsage(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers_admin_plugins.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(b)
	if strings.Contains(text, "h.pluginMgr") || strings.Contains(text, "h.pluginPayMeth") {
		t.Fatalf("admin plugins handler must not use h.pluginMgr/h.pluginPayMeth directly: %s", filepath.Base(path))
	}
}

func TestHandlerStructsNoLegacyPluginDepsFields(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(b)
	if strings.Contains(text, "PluginMgr") || strings.Contains(text, "PluginPayMeth") || strings.Contains(text, "pluginMgr") || strings.Contains(text, "pluginPayMeth") || strings.Contains(text, "PluginDir") || strings.Contains(text, "PluginPass") || strings.Contains(text, "pluginDir") || strings.Contains(text, "pluginPass") {
		t.Fatalf("handler structs must not keep legacy plugin manager/payment repo fields: %s", filepath.Base(path))
	}
}

func TestHandlerStructsNoLegacySettingsRepoField(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(b)
	if strings.Contains(text, "Settings      appports.SettingsRepository") ||
		strings.Contains(text, "settings      appports.SettingsRepository") {
		t.Fatalf("handler structs must not keep direct settings repository fields: %s", filepath.Base(path))
	}
}

func TestHandlerStructsNoLegacyDirectBusinessRepoFields(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(b)
	for _, legacy := range []string{
		"OrderItems    appports.OrderItemRepository",
		"Users         appports.UserRepository",
		"OrderRepo     appports.OrderRepository",
		"VPSRepo       appports.VPSRepository",
		"Payments      appports.PaymentRepository",
		"Permissions   appports.PermissionRepository",
		"Uploads       appports.UploadRepository",
		"ResetTickets  appports.PasswordResetTicketRepository",
		"Broker        *sse.Broker",
		"orderItems    appports.OrderItemRepository",
		"users         appports.UserRepository",
		"orderRepo     appports.OrderRepository",
		"vpsRepo       appports.VPSRepository",
		"payments      appports.PaymentRepository",
		"permissions   appports.PermissionRepository",
		"uploads       appports.UploadRepository",
		"resetTickets  appports.PasswordResetTicketRepository",
		"broker        *sse.Broker",
		"StatusSvc         *appsystemstatus.Service",
		"statusSvc         *appsystemstatus.Service",
		"UploadSvc         *appupload.Service",
		"uploadSvc         *appupload.Service",
		"ReportSvc         *appreport.Service",
		"reportSvc         *appreport.Service",
		"Integration       *appintegration.Service",
		"integration       *appintegration.Service",
		"AuthSvc           *appauth.Service",
		"authSvc           *appauth.Service",
		"OrderSvc          *apporder.Service",
		"orderSvc          *apporder.Service",
		"VPSSvc            *appvps.Service",
		"vpsSvc            *appvps.Service",
	} {
		if strings.Contains(text, legacy) {
			t.Fatalf("handler structs must not keep legacy direct business repo field: %q", legacy)
		}
	}
}

func TestAuthSecurityHandlerNoDirectResetTicketRepoUsage(t *testing.T) {
	root := projectRoot(t)
	path := filepath.Join(root, "internal", "adapter", "http", "handlers_auth_security.go")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(b), "h.resetTickets") {
		t.Fatalf("auth security handler must not use h.resetTickets directly: %s", filepath.Base(path))
	}
}

func TestHTTPHandlersNoDirectSettingsRepoUsageOutsideUtilities(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "adapter", "http")
	var violations []string
	err := filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if filepath.Base(path) == "handlers_utilities.go" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "h.settings.") {
			violations = append(violations, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk http handlers: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("direct settings repo usage outside utilities is forbidden: %s", strings.Join(violations, ", "))
	}
}

func TestAdminOrderAndDebugHandlersNoDirectEventOrAutomationLogRepo(t *testing.T) {
	root := projectRoot(t)
	files := []string{
		filepath.Join(root, "internal", "adapter", "http", "handlers_admin_orders_tickets.go"),
		filepath.Join(root, "internal", "adapter", "http", "handlers_admin_settings_automation.go"),
	}
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(b)
		if strings.Contains(text, "h.eventsRepo") || strings.Contains(text, "h.automationLog") {
			t.Fatalf("direct events/automation log repo usage is forbidden: %s", filepath.Base(path))
		}
	}
}

func TestErrorsNewOnlyAllowedInDomainErrorsGo(t *testing.T) {
	root := projectRoot(t)
	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if rel == "internal/domain/errors.go" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		needle := "errors" + ".New("
		if strings.Contains(string(b), needle) {
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk project: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("errors.New is forbidden outside internal/domain/errors.go: %s", strings.Join(violations, ", "))
	}
}

func TestShouldBindJSONOnlyAllowedInHTTPValidator(t *testing.T) {
	root := projectRoot(t)
	target := filepath.Join(root, "internal", "adapter", "http")
	var violations []string
	err := filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if filepath.Base(path) == "validator.go" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "ShouldBindJSON(") {
			rel, rerr := filepath.Rel(target, path)
			if rerr != nil {
				rel = path
			}
			violations = append(violations, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk http adapter: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("ShouldBindJSON is forbidden outside validator.go: %s", strings.Join(violations, ", "))
	}
}

func TestGoFileNameConventionInCoreDirs(t *testing.T) {
	root := projectRoot(t)
	dirs := []string{
		filepath.Join(root, "internal", "adapter", "http"),
		filepath.Join(root, "internal", "adapter", "repo", "core"),
		filepath.Join(root, "internal", "adapter", "plugins", "core"),
		filepath.Join(root, "internal", "adapter", "plugins", "automation"),
		filepath.Join(root, "internal", "adapter", "plugins", "payment"),
		filepath.Join(root, "internal", "adapter", "plugins", "realname"),
		filepath.Join(root, "internal", "domain"),
	}
	rule := regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*(_test)?\.go$`)
	var violations []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			if !rule.MatchString(d.Name()) {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					rel = path
				}
				violations = append(violations, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("go file name must be snake_case in core dirs: %s", strings.Join(violations, ", "))
	}
}

func TestNoDeprecatedAdapterImportPaths(t *testing.T) {
	root := projectRoot(t)
	deprecated := []string{
		"xiaoheiplay/internal/adapter/automation",
		"xiaoheiplay/internal/adapter/payment/plugin",
		"xiaoheiplay/internal/adapter/repo",
		"xiaoheiplay/internal/adapter/plugins",
	}
	allowed := map[string]bool{
		"xiaoheiplay/internal/adapter/repo/core":          true,
		"xiaoheiplay/internal/adapter/plugins/core":       true,
		"xiaoheiplay/internal/adapter/plugins/automation": true,
		"xiaoheiplay/internal/adapter/plugins/payment":    true,
		"xiaoheiplay/internal/adapter/plugins/realname":   true,
	}
	fset := token.NewFileSet()
	var violations []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			if allowed[importPath] {
				continue
			}
			for _, rule := range deprecated {
				if importPath == rule || strings.HasPrefix(importPath, rule+"/") {
					rel, rerr := filepath.Rel(root, path)
					if rerr != nil {
						rel = path
					}
					violations = append(violations, filepath.ToSlash(rel)+": "+importPath)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk project: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("deprecated adapter import path found: %s", strings.Join(violations, ", "))
	}
}

func TestDeprecatedAdapterDirsMustNotContainGoFiles(t *testing.T) {
	root := projectRoot(t)
	dirs := []string{
		filepath.Join(root, "internal", "adapter", "automation"),
		filepath.Join(root, "internal", "adapter", "payment", "plugin"),
	}
	var violations []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			violations = append(violations, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("deprecated adapter dirs must stay empty of go files: %s", strings.Join(violations, ", "))
	}
}

func TestRepoCoreDeprecatedMixedFilesMustNotReappear(t *testing.T) {
	root := projectRoot(t)
	repoCore := filepath.Join(root, "internal", "adapter", "repo", "core")
	deprecated := []string{
		"gorm_repo_catalog.go",
		"gorm_repo_content_cms_upload.go",
		"gorm_repo_ticket_notification.go",
		"gorm_repo_scan_models.go",
		"migrate_models_ops.go",
	}
	var violations []string
	for _, name := range deprecated {
		path := filepath.Join(repoCore, name)
		if _, err := os.Stat(path); err == nil {
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				rel = path
			}
			violations = append(violations, filepath.ToSlash(rel))
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("deprecated mixed repo files must not reappear: %s", strings.Join(violations, ", "))
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func collectImportViolations(t *testing.T, dir string, forbidden []string) []string {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("stat %s: %v", dir, err)
	}

	var violations []string
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return parseErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			for _, rule := range forbidden {
				if strings.Contains(importPath, rule) {
					violations = append(violations, rel+": "+importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return violations
}

func dropViolationsWithPrefix(in []string, prefix string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func collectFileImportViolations(t *testing.T, files []string, forbidden []string) []string {
	t.Helper()
	fset := token.NewFileSet()
	violations := make([]string, 0)
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, "\"")
			for _, rule := range forbidden {
				if strings.Contains(importPath, rule) {
					violations = append(violations, filepath.Base(path)+": "+importPath)
				}
			}
		}
	}
	return violations
}
