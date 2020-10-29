package reader

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
)

var _ Reader = &KustomizeReader{}

type Reader interface {
	Read(path string) (rm resmap.ResMap, err error)
}

type KustomizeReader struct{}

func (kr *KustomizeReader) Read(path string) (rm resmap.ResMap, err error) {
	fSys := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()

	k := krusty.MakeKustomizer(fSys, opts)

	rm, err = k.Run(path)
	if err != nil {
		return nil, fmt.Errorf("Kustomizer Run for path '%s' failed: %s", path, err)
	}

	return rm, nil
}
