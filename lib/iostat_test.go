package mpiostat

import (
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"
)

type fileStat struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     syscall.Stat_t
	os.FileInfo
}

func (fs *fileStat) Name() string       { return fs.name }
func (fs *fileStat) IsDir() bool        { return true }
func (fs *fileStat) Size() int64        { return fs.size }
func (fs *fileStat) Mode() os.FileMode  { return fs.mode }
func (fs *fileStat) ModTime() time.Time { return fs.modTime }
func (fs *fileStat) Sys() interface{}   { return &fs.sys }

func TestFetchMetrics(t *testing.T) {
	iostat := &IostatPlugin{}

	ret, err := iostat.FetchMetrics()

	if err != nil {
		t.Errorf("FetchMetrics returns error: %s", err)
	}

	if !(0 < ret["seconds"]) {
		// t.Errorf("FetchMetrics doesn't return a value greater than 0")
	}
}

// TODO: Add test for kernel 4.19+
func TestFormatDiskstats(t *testing.T) {
	iostat := &IostatPlugin{}
	stats := `   7       0 loop0 12330 0 26704 960 0 0 0 0 0 68 720
   7       1 loop1 278 0 2590 48 0 0 0 0 0 8 28
 252       0 vda 62695 0 2880751 26352 1383415 166185 10725792 396176 0 25204 301208
 252       1 vda1 62667 0 2878639 26344 979340 166185 10725792 360728 0 25196 301204
 252      16 vdb 526 62 12008 96 964 10253 89736 556 0 160 280
`
	expected := [][]string{
		{"7", "0", "loop0", "12330", "0", "26704", "960", "0", "0", "0", "0", "0", "68", "720"},
		{"7", "1", "loop1", "278", "0", "2590", "48", "0", "0", "0", "0", "0", "8", "28"},
		{"252", "0", "vda", "62695", "0", "2880751", "26352", "1383415", "166185", "10725792", "396176", "0", "25204", "301208"},
		{"252", "1", "vda1", "62667", "0", "2878639", "26344", "979340", "166185", "10725792", "360728", "0", "25196", "301204"},
		{"252", "16", "vdb", "526", "62", "12008", "96", "964", "10253", "89736", "556", "0", "160", "280"},
	}

	got := iostat.formatDiskstats(stats)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("formatDiskstats doesn't format diskstats as expected")
	}
}

// TODO: Test for more than 2 disks.
func TestParseStats(t *testing.T) {
	iostat := &IostatPlugin{}
	stats := [][]string{
		{"252", "0", "vda", "62695", "0", "2880751", "26352", "1383415", "166185", "10725792", "396176", "0", "25204", "301208"},
	}
	got := make(map[string]float64)

	expected := map[string]float64{
		"request.vda.reads":   (62695.0 / 60),
		"merge.vda.reads":     (0.0 / 60),
		"sector.vda.read":     (2880751.0 / 60),
		"time.vda.read":       (26352.0 / 60),
		"request.vda.writes":  (1383415.0 / 60),
		"merge.vda.writes":    (166185.0 / 60),
		"sector.vda.written":  (10725792.0 / 60),
		"time.vda.write":      (396176.0 / 60),
		"inprogress.vda.io":   0.0,
		"time.vda.io":         (25204.0 / 60),
		"time.vda.ioWeighted": (301208.0 / 60),
		/*
			"request.vda.discards": (0.0 / 60),
			"merge.vda.discards":   (0.0 / 60),
			"sector.vda.Discarded": (0.0 / 60),
			"time.vda.discard":     (0.0 / 60),
		*/
	}
	for _, disk := range stats {
		iostat.parseStats(disk[2], disk, got)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("parseStats doesn't parse diskstats as expected")
	}
}

func TestAnalyzeBlockdevices(t *testing.T) {
	iostat := &IostatPlugin{}
	devices := []fileStat{
		fileStat{
			name:    "vda",
			size:    0,
			mode:    os.ModeSymlink,
			modTime: time.Unix(63671245515+62135638488, 0x178350f5&(1<<30-1)),
			/*
				modTime: time.Time{
				wall: 0x178350f5,
					ext:  63671245515,
					loc:  (*time.Location)(0x55f120),
				},
			*/
			sys: syscall.Stat_t{
				Dev:     0x16,
				Ino:     0x3904,
				Nlink:   0x1,
				Mode:    0xa1ff,
				Uid:     0x0,
				Gid:     0x0,
				X__pad0: 0,
				Rdev:    0x0,
				Size:    0,
				Blksize: 4096,
				Blocks:  0,
				Atim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394481909,
				},
				Mtim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394481909,
				},
				Ctim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394481909,
				},
				X__unused: [3]int64{0, 0, 0},
			},
		},
		fileStat{
			name:    "loop0",
			size:    0,
			mode:    os.ModeSymlink,
			modTime: time.Unix(63671245515+62135638488, 0x178350f5&(1<<30-1)),
			/*
				modTime: time.Time{
					wall: 0x1782b02b,
					ext:  63671245515,
					loc:  (*time.Location)(0x55f120),
				},
			*/
			sys: syscall.Stat_t{
				Dev:     0x16,
				Ino:     0x2de7,
				Nlink:   0x1,
				Mode:    0xa1ff,
				Uid:     0x0,
				Gid:     0x0,
				X__pad0: 0,
				Rdev:    0x0,
				Size:    0,
				Blksize: 4096,
				Blocks:  0,
				Atim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394440747,
				},
				Mtim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394440747,
				},
				Ctim: syscall.Timespec{
					Sec:  1535648715,
					Nsec: 394440747,
				},
				X__unused: [3]int64{0, 0, 0},
			},
		},
	}
	expected := map[string]bool{
		"vda":   true,
		"loop0": false,
	}

	devices_os := []os.FileInfo{}
	for i, _ := range devices {
		devices_os = append(devices_os, os.FileInfo(&devices[i]))
	}
	got, err := iostat.analyzeBlockdevices(devices_os)
	if err != nil {
		t.Errorf("analyzaBlockDevices returns error: %s", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("analyzeBlockdevices doesn't analyze diskstats as expected")
	}
}
