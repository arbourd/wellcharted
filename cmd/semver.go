package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/Masterminds/semver"
	"github.com/mitchellh/cli"
	"gopkg.in/src-d/go-git.v4"
	"k8s.io/helm/pkg/chartutil"
)

// Bump is the command for `semver bump`
type Bump struct {
	UI cli.Ui
}

func (c *Bump) Run(args []string) int {
	if len(args) < 1 {
		c.UI.Error(fmt.Sprintf("wellcharted: compare requires a path.\n\n%s", c.Help()))
		return 1
	}

	dir := args[0]
	path := filepath.Join(dir, "Chart.yaml")

	_, err := chartutil.IsChartDir(dir)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: %s", err))
		return 1
	}

	chart, err := chartutil.LoadChartfile(path)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: %s", err))
		return 1
	}
	v, err := semver.NewVersion(chart.GetVersion())
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: error parsing chart version %s", err))
		return 1
	}
	version := v.IncPatch()
	chart.Version = version.String()

	err = chartutil.SaveChartfile(path, chart)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: error saving chart: %s", err))
		return 1
	}

	return 0
}

func (h *Bump) Synopsis() string {
	return "Bumps a charts semantic version"
}

func (*Bump) Help() string {
	return `
Usage: wellcharted semver bump [path]

	Bumps the chart's semantic version located in path.
`
}

// Compare is the command for `semver compare`
type Compare struct {
	UI cli.Ui
}

func (c *Compare) Run(args []string) int {
	if len(args) < 1 {
		c.UI.Error(fmt.Sprintf("wellcharted: compare requires a path.\n\n%s", c.Help()))
		return 1
	}
	cwd, _ := filepath.Abs("")
	dir := args[0]
	path := filepath.Join(dir, "Chart.yaml")

	_, err := chartutil.IsChartDir(dir)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: %s", err))
		return 1
	}

	yaml, err := remoteChartYAML(cwd, path)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: unable to get remote Chart.yaml: %s", err))
		return 1
	}

	// Exit early if no YAML to compare
	if len(yaml) == 0 {
		c.UI.Info(fmt.Sprintf("Unable to find %s on master. New chart detected.", path))
		return 0
	}

	// Create file to write the chart to
	tmpfile, err := ioutil.TempFile("", "Chart.yaml")
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: unable to create remote Chart.yaml: %s", err))
		return 1
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(yaml)); err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: unable to create remote Chart.yaml: %s", err))
		return 1
	}
	if err := tmpfile.Close(); err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: unable to create remote Chart.yaml: %s", err))
		return 1
	}

	// remote Chart.yaml
	v2chart, err := chartutil.LoadChartfile(tmpfile.Name())
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: %s", err))
		return 1
	}
	v2, err := semver.NewVersion(v2chart.GetVersion())
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: error parsing chart version %s", err))
		return 1
	}

	// local Chart.yaml
	v1chart, err := chartutil.LoadChartfile(path)
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: %s", err))
		return 1
	}
	v1, err := semver.NewVersion(v1chart.GetVersion())
	if err != nil {
		c.UI.Error(fmt.Sprintf("wellcharted: error parsing chart version %s", err))
		return 1
	}

	if !v1.GreaterThan(v2) {
		c.UI.Error(fmt.Sprintf("Version %s for %s is not greater than %s on master.", v1, dir, v2))
		return 1
	}

	c.UI.Info(fmt.Sprintf("New version %s for %s.", v1, dir))
	return 0
}

func remoteChartYAML(cwd string, path string) (string, error) {
	// Open .git
	r, err := git.PlainOpen(cwd)
	if err != nil {
		return "", err
	}

	// Get master revision hash
	h, _ := r.ResolveRevision(plumbing.Revision("refs/remotes/origin/master"))
	if err != nil {
		return "", err
	}

	// Get master commit obj
	commit, err := r.CommitObject(*h)
	if err != nil {
		return "", err
	}

	// Get v2 chart file
	file, err := commit.File(path)
	if err != nil {
		// Return nothing if file isn't found, Chart is new
		if err == object.ErrFileNotFound {
			return "", nil
		}

		return "", err
	}
	contents, err := file.Contents()
	if err != nil {
		return "", err
	}

	return contents, nil
}

func (h *Compare) Synopsis() string {
	return "Compares chart's semantic version against master chart"
}

func (*Compare) Help() string {
	return `
Usage: wellcharted semver Compare [path]

	Compares the chart in the current git branch against
	the master branch. Returns 0 if current chart's semantic
	version is greater than the master branch chart.
`
}
