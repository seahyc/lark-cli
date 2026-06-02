package cmd

import "github.com/yjwong/lark-cli/internal/api"

// nameLookupFunc resolves a single open_id to a display name. Returns ("", err)
// when the lookup fails so the caller can fall back to the open_id.
type nameLookupFunc func(openID string) (string, error)

// nameResolver resolves user open_ids to display names, caching results within a
// single CLI invocation to avoid N redundant contact-API calls and rate limits.
// A failed lookup is cached as the open_id itself so we don't retry it.
type nameResolver struct {
	cache  map[string]string
	lookup nameLookupFunc
}

// newNameResolver builds a resolver backed by the contacts API on the given client.
func newNameResolver(client *api.Client) *nameResolver {
	return &nameResolver{
		cache: make(map[string]string),
		lookup: func(openID string) (string, error) {
			u, err := client.GetUser(openID, "open_id")
			if err != nil {
				return "", err
			}
			if u == nil || u.Name == "" {
				return "", nil
			}
			return u.Name, nil
		},
	}
}

// newNameResolverWithLookup builds a resolver with an injected lookup func (for tests).
func newNameResolverWithLookup(lookup nameLookupFunc) *nameResolver {
	return &nameResolver{
		cache:  make(map[string]string),
		lookup: lookup,
	}
}

// preset seeds the cache with a known open_id -> name mapping (e.g. the DM
// counterpart, or names already present on mention objects), avoiding a lookup.
func (r *nameResolver) preset(openID, name string) {
	if openID == "" || name == "" {
		return
	}
	if _, ok := r.cache[openID]; !ok {
		r.cache[openID] = name
	}
}

// resolve returns the display name for an open_id, falling back to the open_id
// itself if the lookup fails or yields nothing. Results are cached.
func (r *nameResolver) resolve(openID string) string {
	if openID == "" {
		return ""
	}
	if name, ok := r.cache[openID]; ok {
		return name
	}
	name, err := r.lookup(openID)
	if err != nil || name == "" {
		// Fall back to the open_id and cache it so we don't retry.
		r.cache[openID] = openID
		return openID
	}
	r.cache[openID] = name
	return name
}

// resolveAll warms the cache for a batch of open_ids in one pass. Order of the
// individual lookups is deterministic only in that each id is resolved once.
func (r *nameResolver) resolveAll(openIDs []string) {
	for _, id := range openIDs {
		r.resolve(id)
	}
}
