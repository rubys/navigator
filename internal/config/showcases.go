package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rubys/navigator/internal/logger"
	"gopkg.in/yaml.v3"
)

// EventInfo represents an individual event within a showcase
type EventInfo struct {
	Name   string `yaml:":name"`
	Date   string `yaml:":date"`
	Locale string `yaml:":locale"`
}

// ShowcaseInfo represents a showcase location with its events
type ShowcaseInfo struct {
	Name   string                `yaml:":name"`
	Region string                `yaml:":region"`
	Date   string                `yaml:":date"`
	Locale string                `yaml:":locale"`
	Logo   string                `yaml:":logo"`
	Events map[string]*EventInfo `yaml:":events"`
}

// Showcases represents the entire showcases.yml structure
type Showcases struct {
	Years    map[string]map[string]*ShowcaseInfo
	Tenants  []*Tenant
	Studios  []string
	Regions  map[string]bool
	filePath string
}

// Tenant represents a single tenant (app instance)
type Tenant struct {
	Owner  string // Studio name
	Region string
	Name   string // Full display name
	Base   string // Base database name (for symlinks)
	Label  string // Unique identifier
	Scope  string // URL path scope
	Logo   string
	Locale string

	// Computed fields
	Year     string
	Studio   string
	Event    string
	DbPath   string
	PumaPort int
}

// LoadShowcases loads and parses the showcases.yml file
func LoadShowcases(path string) (*Showcases, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read showcases file: %w", err)
	}

	// Parse YAML into raw structure
	var raw map[string]map[string]*ShowcaseInfo
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse showcases YAML: %w", err)
	}

	s := &Showcases{
		Years:    raw,
		Regions:  make(map[string]bool),
		filePath: path,
	}

	// Add index tenant
	s.Tenants = append(s.Tenants, &Tenant{
		Owner: "index",
		Name:  "index",
		Label: "index",
		Scope: "",
	})

	// Process all showcases
	studiosMap := make(map[string]bool)

	for year, yearData := range raw {
		for studioKey, info := range yearData {
			studiosMap[studioKey] = true

			if info.Region != "" {
				s.Regions[info.Region] = true
			}

			if info.Events != nil {
				// Multiple events for this studio/year
				logger.WithFields(map[string]interface{}{
					"year":   year,
					"studio": studioKey,
					"events": len(info.Events),
				}).Debug("Found multiple events for studio/year")
				for eventKey, eventInfo := range info.Events {
					tenant := &Tenant{
						Owner:  info.Name,
						Region: info.Region,
						Name:   fmt.Sprintf("%s - %s", info.Name, eventInfo.Name),
						Base:   fmt.Sprintf("%s-%s", year, studioKey),
						Label:  fmt.Sprintf("%s-%s-%s", year, studioKey, eventKey),
						Scope:  fmt.Sprintf("%s/%s/%s", year, studioKey, eventKey),
						Logo:   info.Logo,
						Locale: coalesce(eventInfo.Locale, info.Locale),
						Year:   year,
						Studio: studioKey,
						Event:  eventKey,
					}
					logger.WithFields(map[string]interface{}{
						"label": tenant.Label,
						"scope": tenant.Scope,
					}).Debug("Created multi-event tenant")
					s.Tenants = append(s.Tenants, tenant)
				}
			} else {
				// Single event for this studio/year
				tenant := &Tenant{
					Owner:  info.Name,
					Region: info.Region,
					Name:   info.Name,
					Label:  fmt.Sprintf("%s-%s", year, studioKey),
					Scope:  fmt.Sprintf("%s/%s", year, studioKey),
					Logo:   info.Logo,
					Locale: info.Locale,
					Year:   year,
					Studio: studioKey,
				}
				s.Tenants = append(s.Tenants, tenant)
			}
		}
	}

	// Convert studios map to sorted slice
	for studio := range studiosMap {
		s.Studios = append(s.Studios, studio)
	}

	// Add demo tenant if in Fly.io environment
	if os.Getenv("FLY_REGION") != "" {
		region := os.Getenv("FLY_REGION")
		s.Tenants = append(s.Tenants, &Tenant{
			Owner:  "Demo",
			Region: region,
			Name:   "demo",
			Label:  "demo",
			Scope:  fmt.Sprintf("regions/%s/demo", region),
			Logo:   "intertwingly.png",
		})
	}

	return s, nil
}

// GetTenant finds a tenant by its label
func (s *Showcases) GetTenant(label string) *Tenant {
	for _, t := range s.Tenants {
		if t.Label == label {
			return t
		}
	}
	return nil
}

// GetTenantByPath finds a tenant by URL path
func (s *Showcases) GetTenantByPath(path string) *Tenant {
	// Remove leading/trailing slashes
	path = strings.Trim(path, "/")

	// Special case for index
	if path == "" || path == "index" {
		return s.GetTenant("index")
	}

	// Debug: log tenant lookup for troubleshooting
	logger.WithFields(map[string]interface{}{
		"path":         path,
		"tenant_count": len(s.Tenants),
	}).Debug("Looking up tenant by path")

	// Find the tenant with the longest matching scope (most specific match)
	var bestMatch *Tenant
	var bestMatchLength int

	for _, t := range s.Tenants {
		if t.Scope != "" && strings.HasPrefix(path, t.Scope) {
			logger.WithFields(map[string]interface{}{
				"tenant": t.Label,
				"scope":  t.Scope,
				"path":   path,
			}).Debug("Tenant scope matches path")
			// Check if this is a better match (longer scope)
			if len(t.Scope) > bestMatchLength {
				bestMatch = t
				bestMatchLength = len(t.Scope)
				logger.WithField("tenant", t.Label).Debug("Found better tenant match")
			}
		}
	}

	if bestMatch != nil {
		logger.WithFields(map[string]interface{}{
			"tenant": bestMatch.Label,
			"scope":  bestMatch.Scope,
			"path":   path,
		}).Debug("Found final tenant match")
	} else {
		logger.WithField("path", path).Debug("No tenant found for path")
	}

	return bestMatch
}

// GetAllTenants returns all configured tenants
func (s *Showcases) GetAllTenants() []*Tenant {
	return s.Tenants
}

// GetTenantsForRegion returns tenants for a specific region
func (s *Showcases) GetTenantsForRegion(region string) []*Tenant {
	var result []*Tenant
	for _, t := range s.Tenants {
		if t.Region == region || t.Region == "" {
			result = append(result, t)
		}
	}
	return result
}

// Watch watches the configuration file for changes
func (s *Showcases) Watch(callback func(*Showcases)) error {
	// This would use fsnotify or similar to watch for file changes
	// For now, just a placeholder
	return nil
}

// Reload reloads the configuration from disk
func (s *Showcases) Reload() error {
	newConfig, err := LoadShowcases(s.filePath)
	if err != nil {
		return err
	}

	// Copy over the new data
	s.Years = newConfig.Years
	s.Tenants = newConfig.Tenants
	s.Studios = newConfig.Studios
	s.Regions = newConfig.Regions

	return nil
}

// coalesce returns the first non-empty string
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// GetDatabasePath returns the database path for a tenant
func (t *Tenant) GetDatabasePath(dbRoot string) string {
	if t.DbPath != "" {
		return t.DbPath
	}

	if t.Owner == "Demo" {
		return filepath.Join("/demo/db", fmt.Sprintf("%s.sqlite3", t.Label))
	}

	return filepath.Join(dbRoot, fmt.Sprintf("%s.sqlite3", t.Label))
}

// GetStoragePath returns the storage path for a tenant
func (t *Tenant) GetStoragePath(storageRoot string) string {
	if t.Owner == "Demo" {
		return filepath.Join("/demo/storage", t.Label)
	}
	return filepath.Join(storageRoot, t.Label)
}

// GetEnvironment returns environment variables for this tenant
func (t *Tenant) GetEnvironment(railsRoot, dbPath, storagePath string) []string {
	env := []string{
		fmt.Sprintf("RAILS_APP_DB=%s", t.Label),
		fmt.Sprintf("RAILS_APP_OWNER=%s", t.Owner),
		fmt.Sprintf("DATABASE_URL=sqlite3://%s", t.GetDatabasePath(dbPath)),
		fmt.Sprintf("RAILS_STORAGE=%s", t.GetStoragePath(storagePath)),
		fmt.Sprintf("RAILS_APP_SCOPE=%s", t.Scope),
		"RAILS_ENV=production",
		"RAILS_PROXY_HOST=localhost:3000",     // Set proxy host for Rails templates
		"RAILS_APP_REDIS=showcase_production", // Redis configuration
	}

	if t.Label == "index" {
		env = append(env, "RAILS_SERVE_STATIC_FILES=true")
	} else {
		logo := t.Logo
		if logo == "" {
			logo = "arthur-murray-logo.gif"
		}
		env = append(env, fmt.Sprintf("SHOWCASE_LOGO=%s", logo))

		locale := t.Locale
		if locale == "" {
			locale = "en_US"
		}
		env = append(env, fmt.Sprintf("RAILS_LOCALE=%s", locale))
	}

	if t.PumaPort > 0 {
		env = append(env, fmt.Sprintf("PORT=%d", t.PumaPort))
	}

	return env
}
