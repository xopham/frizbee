//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ghactions provides command-line utilities to work with GitHub Actions.
package ghactions

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/google/go-github/v56/github"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/frizbee/pkg/config"
	"github.com/stacklok/frizbee/pkg/ghactions"
	"github.com/stacklok/frizbee/pkg/utils"
	cliutils "github.com/stacklok/frizbee/pkg/utils/cli"
)

type replacer struct {
	cliutils.Replacer
	ghcli *github.Client
}

func (r *replacer) do(ctx context.Context, cfg *config.Config) error {
	basedir := filepath.Dir(r.Dir)
	base := filepath.Base(r.Dir)
	bfs := osfs.New(basedir, osfs.WithBoundOS())

	outfiles := map[string]string{}
	modified := false

	err := ghactions.TraverseGitHubActionWorkflows(bfs, base, func(path string, wflow *yaml.Node) error {
		r.Logf("Processing %s\n", path)
		m, err := ghactions.ModifyReferencesInYAML(ctx, r.ghcli, wflow, &cfg.GHActions)
		if err != nil {
			return fmt.Errorf("failed to process YAML file %s: %w", path, err)
		}

		modified = modified || m

		buf, err := utils.YAMLToBuffer(wflow)
		if err != nil {
			return fmt.Errorf("failed to convert YAML to buffer: %w", err)
		}

		if m {
			r.Logf("Modified %s\n", path)
			outfiles[path] = buf.String()
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := r.ProcessOutput(bfs, outfiles); err != nil {
		return err
	}

	if r.ErrOnModified && modified {
		return fmt.Errorf("modified files")
	}

	return nil
}
