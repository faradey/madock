package dockertransform

import "testing"

type recorder struct{ seen string }

func (r *recorder) TransformCompose(_, content string) string {
	r.seen = RenderingProject()
	return content
}
func (r *recorder) TransformDockerfile(_, content string) string { return content }

// A transformer is told whose file it holds, and the name does not outlive the
// call: project:clone renders the clone's files from inside the source.
func TestATransformerIsToldWhoseFileItHolds(t *testing.T) {
	r := &recorder{}
	SetDockerTransformer(r)
	defer SetDockerTransformer(nil)

	ApplyProjectComposeTransform("clone", "docker-compose.yml", "")
	if r.seen != "clone" {
		t.Errorf("RenderingProject during the transform = %q, want clone", r.seen)
	}
	if got := RenderingProject(); got != "" {
		t.Errorf("RenderingProject after the transform = %q, want empty", got)
	}
}
