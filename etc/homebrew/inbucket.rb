require "language/go"

class Inbucket < Formula
  desc "Disposable webmail service with built in SMTP, POP3, REST servers"
  homepage "http://www.inbucket.org/"
  url "https://github.com/jhillyerd/inbucket.git", :tag => "1.1.0-rc3"

  head "https://github.com/jhillyerd/inbucket.git", :branch => "develop"

  devel do
    url "https://github.com/jhillyerd/inbucket.git", :branch => "master"
    version "1.1.1-alpha"
  end

  def log_path
    var/"log/inbucket.log"
  end

  depends_on "go" => :build

  go_resource "github.com/goods/httpbuf" do
    url "https://github.com/goods/httpbuf.git",
      :revision => "5709e9bb814c932e48b6737e1cf214a6522453a2"
  end

  go_resource "github.com/gorilla/context" do
    url "https://github.com/gorilla/context.git",
      :revision => "1ea25387ff6f684839d82767c1733ff4d4d15d0a"
  end

  go_resource "github.com/gorilla/mux" do
    url "https://github.com/gorilla/mux.git",
      :revision => "acf3be1b335c8ce30b2c8d51300984666f0ceefa"
  end

  go_resource "github.com/gorilla/securecookie" do
    url "https://github.com/gorilla/securecookie.git",
      :revision => "8dacca26977607e637262eb66b15b7d39f2d3009"
  end

  go_resource "github.com/gorilla/sessions" do
    url "https://github.com/gorilla/sessions.git",
      :revision => "8cd570d8b4ed84b18bca9d8c3ae2db55885ccd8b"
  end

  go_resource "github.com/jaytaylor/html2text" do
    url "https://github.com/jaytaylor/html2text.git",
      :revision => "4936b6c2ae7f89f5eeba2473c31fd27ea8f11068"
  end

  go_resource "github.com/jhillyerd/go.enmime" do
    url "https://github.com/jhillyerd/go.enmime.git",
      :revision => "3ea281bf3e00864f4afe2f9a6911af164438b581"
  end

  go_resource "github.com/robfig/config" do
    url "https://github.com/robfig/config.git",
      :revision => "0f78529c8c7e3e9a25f15876532ecbc07c7d99e6"
  end

  go_resource "golang.org/x/net" do
    url "https://go.googlesource.com/net.git",
      :revision => "3e5cd1ed149001198e582f9d3f5bfd564cde2896"
  end

  go_resource "golang.org/x/sys" do
    url "https://go.googlesource.com/sys.git",
      :revision => "7a56174f0086b32866ebd746a794417edbc678a1"
  end

  go_resource "golang.org/x/text" do
    url "https://go.googlesource.com/text.git",
      :revision => "a71fd10341b064c10f4a81ceac72bcf70f26ea34"
  end

  def install
    package = "github.com/jhillyerd/inbucket"
    contents = Dir["{*,.git,.gitignore}"]
    gopath = buildpath/"gopath"
    (gopath/"src/#{package}").install contents

    ENV["GOPATH"] = gopath
    ENV.prepend_create_path "PATH", gopath/"bin"

    Language::Go.stage_deps resources, gopath/"src"

    cd gopath/"src/#{package}" do
      system "go", "build"
      bin.install "inbucket"
      pkgshare.install "themes"
      inreplace "etc/homebrew/inbucket.conf" do |s|
        # We want the config to use non-versioned paths
        s.gsub!(/{{HOMEBREW_PREFIX}}/, HOMEBREW_PREFIX)
      end
      etc.install "etc/homebrew/inbucket.conf"
    end
  end

  def caveats; <<-EOS.undent
    By default, inbucket listens on the following TCP ports:
      0.0.0.0:2500 - SMTP
      0.0.0.0:1100 - POP3
      0.0.0.0:9000 - HTTP

    You may change these ports by editing #{etc}/inbucket.conf

    Once inbucket has started, access its web interface at:
      http://localhost:9000/
    EOS
  end

  test do
    system "#{bin}/inbucket", "-help"
  end

  plist_options :startup => "true"

  def plist; <<-EOS.undent
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
      <key>Label</key>
      <string>#{plist_name}</string>
      <key>RunAtLoad</key>
      <true/>
      <key>KeepAlive</key>
      <true/>
      <key>WorkingDirectory</key>
      <string>#{HOMEBREW_PREFIX}</string>
      <key>ProgramArguments</key>
      <array>
        <string>#{opt_bin}/inbucket</string>
        <string>#{etc}/inbucket.conf</string>
      </array>
      <key>StandardErrorPath</key>
      <string>#{log_path}</string>
      <key>EnvironmentVariables</key>
      <dict>
        <key>LANG</key>
        <string>en_US.UTF-8</string>
      </dict>
    </dict>
    </plist>
    EOS
  end
end
