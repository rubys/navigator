//go:build unix

package process

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"

	"log/slog"
)

// SysCredential is a type alias for syscall.Credential on Unix systems
type SysCredential = syscall.Credential

// GetUserCredentials looks up a user by name and returns syscall credentials
// Returns nil if username is empty or user doesn't exist
func GetUserCredentials(username, groupname string) (*SysCredential, error) {
	if username == "" {
		return nil, nil
	}

	u, err := user.Lookup(username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s: %w", username, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid uid for user %s: %w", username, err)
	}

	// Determine GID
	var gid uint64
	if groupname != "" {
		// Look up specified group
		g, err := user.LookupGroup(groupname)
		if err != nil {
			return nil, fmt.Errorf("group not found: %s: %w", groupname, err)
		}
		gid, err = strconv.ParseUint(g.Gid, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid gid for group %s: %w", groupname, err)
		}
	} else {
		// Use user's primary group
		gid, err = strconv.ParseUint(u.Gid, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid gid for user %s: %w", username, err)
		}
	}

	slog.Debug("User credentials resolved",
		"username", username,
		"uid", uid,
		"groupname", groupname,
		"gid", gid)

	return &syscall.Credential{
		Uid: uint32(uid),
		Gid: uint32(gid),
	}, nil
}
