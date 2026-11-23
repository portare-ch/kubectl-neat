# Homebrew formula. This repository is its own tap:
#   brew tap portare-ch/kubectl-neat https://github.com/portare-ch/kubectl-neat
#   brew install kubectl-neat
# To release a new version, push the tag and update `tag:` below.
class KubectlNeat < Formula
  desc "Remove clutter from Kubernetes manifests to make them more readable"
  homepage "https://github.com/portare-ch/kubectl-neat"
  url "https://github.com/portare-ch/kubectl-neat.git",
      tag: "v2.1.0"
  license "Apache-2.0"
  head "https://github.com/portare-ch/kubectl-neat.git", branch: "master"

  depends_on "go" => :build

  def install
    ldflags = "-X github.com/itaysk/kubectl-neat/cmd.Version=#{version}"
    system "go", "build", *std_go_args(ldflags:)
    bin.install_symlink "kubectl-neat" => "kneat"
  end

  test do
    pod = <<~YAML
      apiVersion: v1
      kind: Pod
      metadata:
        name: mypod
        namespace: default
        uid: 0b6e5c1e-5f43-4c1e-9b8f-2d0c4f6a7e01
        creationTimestamp: "2026-01-01T00:00:00Z"
      spec:
        containers:
        - name: app
          image: nginx
          terminationMessagePolicy: File
        nodeName: node-1
      status:
        phase: Running
    YAML
    out = pipe_output("#{bin}/kneat", pod, 0)
    assert_match "name: mypod", out
    refute_match "uid:", out
    refute_match "nodeName", out
    refute_match "status:", out
    assert_match version.to_s, shell_output("#{bin}/kubectl-neat version")
  end
end
