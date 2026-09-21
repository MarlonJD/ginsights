class Ginsights < Formula
  desc "GitHub-style local repository insights as a single Go binary"
  homepage "https://github.com/MarlonJD/ginsights"
  url "https://github.com/MarlonJD/ginsights/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "2e3c3458d918ca5eedb3ab4024d27855ba95939581d453274cd065cc00c78b1b"
  head "https://github.com/MarlonJD/ginsights.git", branch: "main"
  license "GPL-3.0-or-later"

  depends_on "go" => :build
  depends_on "git"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/ginsights"
  end

  test do
    system "#{bin}/ginsights", "help"
  end
end
