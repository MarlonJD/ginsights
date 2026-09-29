package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const liveStatus = "Live updates enabled."
const liveErrorStatus = "Live update failed. Showing the last successful report; retrying automatically."

func liveHTML(html, etag string, interval time.Duration) string {
	status := fmt.Sprintf(` <span id="live-status" role="status">%s</span> Checks every %s.`, liveStatus, interval)
	html = strings.Replace(html, "</footer>", status+"</footer>", 1)
	script := fmt.Sprintf(`<script>
(() => {
  const etag = %s;
  const interval = Math.min(%d, 2147483647);
  const status = document.getElementById("live-status");
  async function check() {
    if (!document.hidden) {
      try {
        const response = await fetch("/data.json", {
          method: "HEAD",
          cache: "no-store",
          headers: {"If-None-Match": etag}
        });
        if (response.status === 200 && response.headers.get("ETag") !== etag) {
          window.location.reload();
          return;
        }
        if (response.status !== 200 && response.status !== 304) throw new Error("Refresh failed");
        status.textContent = %s;
      } catch {
        status.textContent = %s;
      }
    }
    window.setTimeout(check, interval);
  }
  window.setTimeout(check, interval);
})();
</script>
`, strconv.Quote(etag), interval.Milliseconds(), strconv.Quote(liveStatus), strconv.Quote(liveErrorStatus))
	return strings.Replace(html, "</body>", script+"</body>", 1)
}
