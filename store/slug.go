package store

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// ValidateSlug enforces: lowercase alphanumeric + hyphens only, 3–50 chars,
// no leading/trailing hyphen, no leading underscore.
func ValidateSlug(slug string) error {
	if len(slug) < 3 {
		return fmt.Errorf("slug must be at least 3 characters long")
	}
	if len(slug) > 50 {
		return fmt.Errorf("slug must be at most 50 characters long")
	}
	if strings.HasPrefix(slug, "_") {
		return fmt.Errorf("slug cannot start with an underscore")
	}
	if strings.HasPrefix(slug, "-") {
		return fmt.Errorf("slug cannot start with a hyphen")
	}
	if strings.HasSuffix(slug, "-") {
		return fmt.Errorf("slug cannot end with a hyphen")
	}
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug may only contain lowercase letters, digits, and hyphens")
	}
	return nil
}

var adjectives = []string{
	"silent", "fuzzy", "bright", "calm", "swift", "golden", "dark", "warm",
	"cool", "bold", "wise", "kind", "flat", "deep", "sharp", "clean",
	"wild", "soft", "fast", "loud", "rare", "tall", "wide", "thin",
}

var nouns = []string{
	"river", "wolf", "stone", "cloud", "flame", "tree", "leaf", "rain",
	"bird", "peak", "tide", "frost", "moon", "star", "lake", "wind",
	"dust", "wave", "reed", "hawk", "bear", "pine", "vale", "mist",
}

// GenerateSlug generates a slug in the format {adjective}-{noun}-{4-digit-number}.
// It retries if the generated slug already exists (as determined by the existing function).
func GenerateSlug(existing func(string) bool) string {
	for {
		adj := adjectives[cryptoRandN(len(adjectives))]
		noun := nouns[cryptoRandN(len(nouns))]
		num := 1000 + cryptoRandN(9000)
		slug := fmt.Sprintf("%s-%s-%d", adj, noun, num)
		if !existing(slug) {
			return slug
		}
	}
}

// cryptoRandN returns a cryptographically random int in [0, n).
func cryptoRandN(n int) int {
	max := big.NewInt(int64(n))
	val, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Extremely unlikely; fall back to 0 rather than panicking.
		return 0
	}
	return int(val.Int64())
}
