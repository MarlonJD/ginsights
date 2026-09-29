class Ginsights < Formula
  desc "GitHub-style local repository insights as a single Go binary"
  homepage "https://github.com/MarlonJD/ginsights"
  url "https://github.com/MarlonJD/ginsights/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "656e280a0837c574e5066451e32410ebbf854c868892f311077aa95a7ff24791"
  license "GPL-3.0-or-later"
  head "https://github.com/MarlonJD/ginsights.git", branch: "main"

  depends_on "go" => :build
  depends_on "git"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/ginsights"
  end

  test do
    system bin/"ginsights", "help"
  end
end
