package safety

import (
	"slices"
	"testing"
)

func TestCheck_Safe(t *testing.T) {
	r := Check("ls -la")
	if r.Level != Safe {
		t.Error("ls -la should be safe")
	}
}

func TestCheck_SafeDevNull(t *testing.T) {
	r := Check("echo test > /dev/null")
	if r.Level != Safe {
		t.Error("/dev/null redirect should be safe")
	}
}

func TestCheck_DangerousDeviceWrite(t *testing.T) {
	r := Check("echo test > /dev/sda")
	if r.Level != Dangerous {
		t.Error("device redirect should be dangerous")
	}
	if !slices.Contains(r.Matches, "write to device") {
		t.Error("should report device write")
	}
}

func TestCheck_WarningSudo(t *testing.T) {
	r := Check("sudo apt-get install vim")
	if r.Level != Warning {
		t.Error("sudo should trigger warning")
	}
	if !slices.Contains(r.Matches, "sudo") {
		t.Error("should report sudo")
	}
}

func TestCheck_WarningChmod(t *testing.T) {
	r := Check("chmod 777 /var/www")
	if r.Level != Warning {
		t.Error("chmod 777 should trigger warning")
	}
}

func TestCheck_DangerousRmRoot(t *testing.T) {
	r := Check("rm -rf / --no-preserve-root")
	if r.Level != Dangerous {
		t.Error("rm -rf / should be dangerous")
	}
	if !slices.Contains(r.Matches, "rm on root") {
		t.Error("should report root rm")
	}
}

func TestCheck_WarningRecursiveRm(t *testing.T) {
	r := Check("rm -rf /tmp/stuff")
	if r.Level != Warning {
		t.Error("recursive rm outside root should be warning")
	}
	if slices.Contains(r.Matches, "rm on root") {
		t.Error("non-root recursive rm should not report root rm")
	}
}

func TestCheck_DangerousMkfs(t *testing.T) {
	r := Check("mkfs.ext4 /dev/sda1")
	if r.Level != Dangerous {
		t.Error("mkfs should be dangerous")
	}
}

func TestCheck_MultipleMatches(t *testing.T) {
	r := Check("sudo rm -rf /tmp/stuff")
	if len(r.Matches) < 2 {
		t.Errorf("should match multiple patterns, got %d", len(r.Matches))
	}
}
