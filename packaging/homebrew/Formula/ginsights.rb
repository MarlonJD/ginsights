class Ginsights < Formula
  desc "GitHub-style local repository insights as a single Go binary"
  homepage "https://github.com/MarlonJD/ginsights"
  url "https://github.com/MarlonJD/ginsights/archive/refs/tags/v0.1.2.tar.gz"
  sha256 "26b346d900389d21f72a568f4b972805bbb99508f194235f4d159ea80f0ba43a"
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
