PRODUCT      := edgestream-scheduler
DEBEMAIL     := Hyperi <edgestream-support@hyperi.io>
CONTENTS     := Hyperi - contact edgestream-support@hyperi.io for support enquiries
BASE_VERSION := 0.1.0

# Get the exact tag on the current commit
GIT_TAG := $(shell git describe --tags --exact-match 2>/dev/null)

# Normalize SOURCE_DATE_EPOCH
EPOCHSEC := $(shell \
  if [ -n "$$SOURCE_DATE_EPOCH" ]; then \
    s="$$SOURCE_DATE_EPOCH"; s=$${s%%.*}; \
    if [ "$$s" -ge 1000000000000000000 ] 2>/dev/null; then s=$$(( s/1000000000 )); \
    elif [ "$$s" -ge 1000000000000000 ] 2>/dev/null; then s=$$(( s/1000000 )); \
    elif [ "$$s" -ge 1000000000000 ] 2>/dev/null; then s=$$(( s/1000 )); \
    fi; echo $$s; \
  else date +%s; fi)

DATE_YYYYMMDD := $(shell date -u -d @$(EPOCHSEC) +%Y%m%d)

# --- VERSION LOGIC ---
ifeq ($(GIT_TAG),)
    # 1. No tag? Use the development string (0.0.1.gitYYYYMMDD.EPOCH)
    VERSION := $(BASE_VERSION).git$(DATE_YYYYMMDD).$(EPOCHSEC)
else
    # 2. Tagged? Strip 'v' or 'dev-' prefixes for the internal Debian version
    # Examples: 'v1.0.4' -> '1.0.4' | 'dev-1.0.4' -> '1.0.4'
    VERSION := $(shell echo $(GIT_TAG) | sed -E 's/^(v|dev-)//')
endif

# RFC2822 date for changelog
DATE := $(shell date -u -d @$(EPOCHSEC) '+%a, %d %b %Y %H:%M:%S +0000')

.EXPORT_ALL_VARIABLES:
GOFLAGS := -mod=readonly  # Ensure we use the Go Cache from your CI step
.DEFAULT_GOAL := all

clean_build:
	rm -rf ../*deb
	rm -rf ./debian/.debhelper ./debian/debhelper-build-stamp
	rm -rf ./debian/edgestream-scheduler.postrm.debhelper ./debian/edgestream-scheduler.substvars
	rm -rf ./debian/files ./debian/edgestream-scheduler

clean: clean_build
.PHONY: clean

create_changelog:
	rm -f ./debian/changelog
	# Use --force-bad-version if your git tags don't strictly follow SemVer
	dch --create --package $(PRODUCT) --newversion $(VERSION) --urgency medium "$(CONTENTS)" --distribution unstable
	sed -i 's/^ -- .*$$/ -- $(DEBEMAIL)  $(DATE)/' ./debian/changelog

build_deb:
	chmod +x "debian/rules"
	# -nc (no clean) can be used here if you want to keep the Go objects between debuild calls
	debuild --no-lintian $(if $(_DEBSIG),-k$(_DEBSIG),) -i -uc -us -b -j4

build: build_deb
.PHONY: build

prepare: create_changelog
.PHONY: prepare

dist: clean prepare build

all: dist
.PHONY: all
