package config

import (
    "bufio"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

var (
    reExport = regexp.MustCompile(`^\s*export\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)\s*$`)
    reAssign = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)\s*$`)
)

// LoadEnv loads simple shell-style env files into process env.
// Supports lines like:
//   export KEY=value
//   KEY=value
// Values may be unquoted, single-quoted, or double-quoted. Simple escapes for \\ and \" in double quotes are handled; single quotes are literal.
func LoadEnv(paths ...string) {
    for _, p := range paths {
        if p == "" { continue }
        if fi, err := os.Stat(p); err != nil || fi.IsDir() { continue }
        f, err := os.Open(p)
        if err != nil { continue }
        scan := bufio.NewScanner(f)
        for scan.Scan() {
            line := strings.TrimSpace(scan.Text())
            if line == "" || strings.HasPrefix(line, "#") { continue }
            var key, val string
            if m := reExport.FindStringSubmatch(line); m != nil {
                key, val = m[1], m[2]
            } else if m := reAssign.FindStringSubmatch(line); m != nil {
                key, val = m[1], m[2]
            } else {
                continue
            }
            val = strings.TrimSpace(val)
            if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") && len(val) >= 2 {
                v := val[1:len(val)-1]
                v = strings.ReplaceAll(v, `\\`, `\`)
                v = strings.ReplaceAll(v, `\"`, `"`)
                os.Setenv(key, v)
            } else if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") && len(val) >= 2 {
                os.Setenv(key, val[1:len(val)-1])
            } else {
                os.Setenv(key, val)
            }
        }
        f.Close()
    }
}

// LoadDefaultEnv loads env from MRP_ENV, ~/.mrp.env, and ./.env (in that order), when present.
func LoadDefaultEnv() {
    if p := strings.TrimSpace(os.Getenv("MRP_ENV")); p != "" {
        LoadEnv(p)
    }
    if home, err := os.UserHomeDir(); err == nil {
        LoadEnv(filepath.Join(home, ".mrp", ".unused")) // don't fail if missing dir
        LoadEnv(filepath.Join(home, ".mrp.env"))
    }
    LoadEnv(".env")
}

