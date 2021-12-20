// This file contains to logic to generate the files for deploying onto a webspace.

package render

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/anexia-it/go.anx.io/pkg/types"
)

func (r *Renderer) GenerateFiles(destPath string) error {
	type pkgFiles struct {
		pkg   *types.Package
		files []string
	}

	allFiles := make([]pkgFiles, 0, len(r.packages)+1)

	for _, pkg := range r.packages {
		allFiles = append(allFiles, pkgFiles{
			pkg:   pkg,
			files: r.filesForPackage(pkg),
		})
	}

	if files, err := r.filesForContent(); err == nil {
		allFiles = append(allFiles, pkgFiles{
			files: files,
		})
	} else {
		return err
	}

	for _, pf := range allFiles {
		pkg := pf.pkg

		for _, file := range pf.files {
			fileExt := path.Ext(file)
			destinationPath := path.Join(destPath, file)

			if pkg != nil {
				destinationPath = path.Join(destPath, pkg.TargetName, file)

				fileAndVersion := strings.SplitN(file, "@", 2)
				fileExt = path.Ext(fileAndVersion[0])
			}

			if fileExtensionNeedsMIMEHack(fileExt) {
				destinationPath = path.Join(destinationPath, "index.html")
			}

			destinationDirectory := path.Dir(destinationPath)
			if err := os.MkdirAll(destinationDirectory, os.ModeDir|0755); err != nil {
				return fmt.Errorf("error creating directory %q: %w", destinationDirectory, err)
			}

			if err := r.generateFile(pkg, destinationPath, file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Renderer) generateFile(pkg *types.Package, dest, file string) error {
	w, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file %q (create|truncate|write): %w", dest, err)
	}

	if err := r.RenderFile(pkg, file, w); err != nil {
		var pkgName string = "<nil>"
		if pkg != nil {
			pkgName = pkg.TargetName
		}

		return fmt.Errorf("error rendering file %q for package %q: %w", file, pkgName, err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("error closing file %q after writing to it: %w", dest, err)
	}

	return nil
}

// On webhosting services like Github Pages we cannot send a custom mime type for our files,
// they only use the file extension for choosing a mime type to send to the browser. Since
// we want our paths to reflect names in repositories (e.g. `README.md`) but render them to HTML,
// we create a directory for those files and place a `index.html` in that.
func fileExtensionNeedsMIMEHack(fileExt string) bool {
	// This array defines the file extensions we do this for.
	mimeTypeHackExtensions := []string{".md"}

	// we need it sorted for fast check if a given extension is in this array
	sort.Strings(mimeTypeHackExtensions)

	idx := sort.SearchStrings(mimeTypeHackExtensions, fileExt)

	if idx >= len(mimeTypeHackExtensions) {
		return false
	}

	return mimeTypeHackExtensions[idx] == fileExt
}
