class Lionclaw < Formula
  desc "Secure, self-hosted personal AI Agent platform"
  homepage "https://github.com/amszh10100-blip/lionclaw"
  url "https://github.com/amszh10100-blip/lionclaw/archive/refs/tags/v2.0.0.tar.gz"
  license "MIT"

  depends_on "go" => :build
  depends_on "sqlite"

  def install
    cd "src" do
      system "go", "build", "-tags", "fts5",
             "-ldflags", "-X main.version=#{version}",
             "-o", bin/"lionclaw", "./cmd/lionclaw"
    end
  end

  test do
    assert_match "lionclaw v#{version}", shell_output("#{bin}/lionclaw version")
  end
end
