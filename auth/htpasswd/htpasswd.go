package htpasswd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/danielkrainas/canaria-api/auth"

	"golang.org/x/crypto/bcrypt"
)

type htpasswd struct {
	entries map[string][]byte
}

func newHTPasswd(rd io.Reader) (*htpasswd, error) {
	entries, err := parseHTPasswd(rd)
	if err != nil {
		return nil, err
	}

	return &htpasswd{entries: entries}, nil
}

func (htpasswd *htpasswd) authenticateUser(username string, password string) error {
	credentials, ok := htpasswd.entries[username]
	if !ok {
		// keep same timing
		bcrypt.CompareHashAndPassword([]byte{}, []byte(password))
		return auth.ErrAuthenticationFailure
	}

	err := bcrypt.CompareHashAndPassword([]byte(credentials), []byte(password))
	if err != nil {
		return auth.ErrAuthenticationFailure
	}

	return nil
}

func parseHTPasswd(rd io.Reader) (map[string][]byte, error) {
	entries := map[string][]byte{}
	scanner := bufio.NewScanner(rd)
	var line int
	for scanner.Scan() {
		line++
		t := strings.TrimSpace(scanner.Text())
		if len(t) < 1 {
			continue
		}

		if t[0] == '#' {
			continue
		}

		i := strings.Index(t, ":")
		if i < 0 || i >= len(t) {
			return nil, fmt.Errorf("htpasswd: invalid entry at line %d: %q", line, scanner.Text())
		}

		entries[t[:i]] = []byte(t[i+1:])
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
