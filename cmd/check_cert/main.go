// Copyright 2020 Adam Chalkley
//
// https://github.com/atc0005/check-cert
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/atc0005/check-cert/internal/certs"
	"github.com/atc0005/check-cert/internal/config"
	"github.com/atc0005/check-cert/internal/netutils"
	"github.com/atc0005/go-nagios"
)

func main() {

	// Start the timer. We'll use this to emit the plugin runtime as a
	// performance data metric.
	pluginStart := time.Now()

	// Set initial "state" as valid, adjust as we go.
	var nagiosExitState = nagios.ExitState{
		LastError:      nil,
		ExitStatusCode: nagios.StateOKExitCode,
	}

	// defer this from the start so it is the last deferred function to run
	defer nagiosExitState.ReturnCheckResults()

	// Setup configuration by parsing user-provided flags.
	cfg, cfgErr := config.New(config.AppType{Plugin: true})
	switch {
	case errors.Is(cfgErr, config.ErrVersionRequested):
		fmt.Println(config.Version())

		return

	case cfgErr != nil:
		// We're using the standalone Err function from rs/zerolog/log as we
		// do not have a working configuration.
		zlog.Err(cfgErr).Msg("Error initializing application")
		nagiosExitState.ServiceOutput = fmt.Sprintf(
			"%s: Error initializing application",
			nagios.StateCRITICALLabel,
		)
		nagiosExitState.AddError(cfgErr)
		nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

		return
	}

	// Collect last minute details just before ending plugin execution.
	defer func(exitState *nagios.ExitState, start time.Time, logger zerolog.Logger) {

		// Record plugin runtime, emit this metric regardless of exit
		// point/cause.
		runtimeMetric := nagios.PerformanceData{
			Label: "time",
			Value: fmt.Sprintf("%dms", time.Since(start).Milliseconds()),
		}
		if err := exitState.AddPerfData(false, runtimeMetric); err != nil {
			zlog.Error().
				Err(err).
				Msg("failed to add time (runtime) performance data metric")
		}

		// Annotate errors (if applicable) with additional context to aid in
		// troubleshooting.
		nagiosExitState.Errors = annotateError(logger, nagiosExitState.Errors...)
	}(&nagiosExitState, pluginStart, cfg.Log)

	if cfg.EmitBranding {
		// If enabled, show application details at end of notification
		nagiosExitState.BrandingCallback = config.Branding("Notification generated by ")
	}

	// Use provided threshold values to calculate the expiration times that
	// should trigger either a WARNING or CRITICAL state.
	now := time.Now().UTC()
	certsExpireAgeWarning := now.AddDate(0, 0, cfg.AgeWarning)
	certsExpireAgeCritical := now.AddDate(0, 0, cfg.AgeCritical)

	nagiosExitState.WarningThreshold = fmt.Sprintf(
		"Expires before %v (%d days)",
		certsExpireAgeWarning.Format(certs.CertValidityDateLayout),
		cfg.AgeWarning,
	)
	nagiosExitState.CriticalThreshold = fmt.Sprintf(
		"Expires before %v (%d days)",
		certsExpireAgeCritical.Format(certs.CertValidityDateLayout),
		cfg.AgeCritical,
	)

	log := cfg.Log.With().
		Str("expected_sans_entries", cfg.SANsEntries.String()).
		Logger()

	var certChain []*x509.Certificate

	var certChainSource string

	// Honor request to parse filename first
	switch {
	case cfg.Filename != "":

		log.Debug().Msg("Attempting to parse certificate file")

		// Anything from the specified file that couldn't be converted to a
		// certificate chain. While likely not of high value by itself,
		// failure to parse a certificate file indicates a likely source of
		// trouble. We consider this scenario to be a CRITICAL state.
		var parseAttemptLeftovers []byte

		var err error
		certChain, parseAttemptLeftovers, err = certs.GetCertsFromFile(cfg.Filename)
		if err != nil {
			log.Error().Err(err).Msg(
				"Error parsing certificates file")

			nagiosExitState.AddError(err)
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: Error parsing certificates file %q",
				nagios.StateCRITICALLabel,
				cfg.Filename,
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			return
		}

		certChainSource = cfg.Filename

		log.Debug().Msg("Certificate file parsed")

		if len(parseAttemptLeftovers) > 0 {
			log.Error().Err(err).Msg(
				"Unknown data encountered while parsing certificates file")

			nagiosExitState.AddError(fmt.Errorf(
				"%d unknown/unparsed bytes remaining at end of cert file %q",
				len(parseAttemptLeftovers),
				cfg.Filename,
			))
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: Unknown data encountered while parsing certificates file %q",
				nagios.StateCRITICALLabel,
				cfg.Filename,
			)

			nagiosExitState.LongServiceOutput = fmt.Sprintf(
				"The following text from the %q certificate file failed to parse"+
					" and is provided here for troubleshooting purposes:%s%s%s",
				cfg.Filename,
				nagios.CheckOutputEOL,
				nagios.CheckOutputEOL,
				string(parseAttemptLeftovers),
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			return
		}

	case cfg.Server != "":

		log.Debug().Msg("Expanding given host pattern in order to obtain IP Address")
		expandedHost, expandErr := netutils.ExpandHost(cfg.Server)
		switch {
		case expandErr != nil:
			log.Error().Err(expandErr).Msg(
				"Error expanding given host pattern")

			nagiosExitState.AddError(expandErr)
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: Error expanding given host pattern %q to target IP Address",
				nagios.StateCRITICALLabel,
				cfg.Server,
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			// no need to go any further, we *want* to exit right away; we don't
			// have a connection to the remote server and there isn't anything
			// further we can do
			return

		// Fail early for IP Ranges. While we could just grab the first
		// expanded IP Address, this may be a potential source of confusion
		// best avoided.
		case expandedHost.Range:
			invalidHostPatternErr := errors.New("invalid host pattern")
			msg := fmt.Sprintf(
				"Given host pattern invalid; " +
					"host pattern is a CIDR or partial IP range",
			)
			log.Error().Err(invalidHostPatternErr).Msg(msg)

			nagiosExitState.AddError(invalidHostPatternErr)
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: %s",
				nagios.StateCRITICALLabel,
				msg,
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			// no need to go any further, we *want* to exit right away; we don't
			// have a connection to the remote server and there isn't anything
			// further we can do
			return

		case len(expandedHost.Expanded) == 0:
			expandHostErr := errors.New("host pattern expansion failed")
			msg := "Error expanding given host value to IP Address"

			log.Error().Err(expandHostErr).Msg(msg)

			nagiosExitState.AddError(expandHostErr)
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: %s",
				nagios.StateCRITICALLabel,
				msg,
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			// no need to go any further, we *want* to exit right away; we don't
			// have a connection to the remote server and there isn't anything
			// further we can do
			return

		case len(expandedHost.Expanded) > 1:

			ipAddrs := zerolog.Arr()
			for _, ip := range expandedHost.Expanded {
				ipAddrs.Str(ip)
			}

			log.Debug().
				Int("num_ip_addresses", len(expandedHost.Expanded)).
				Array("ip_addresses", ipAddrs).
				Msg("Multiple IP Addresses resolved from given host pattern")
			log.Debug().Msg("Using first IP Address, ignoring others")
		}

		// Grab first IP Address from the resolved collection. We'll
		// explicitly use it for cert retrieval and note it in the report
		// output.
		ipAddr := expandedHost.Expanded[0]

		// Server Name Indication (SNI) support is used to request a specific
		// certificate chain from a remote server.
		//
		// We use the value specified by the `server` flag to open a
		// connection to the remote server. If available, we use the DNS Name
		// value specified by the `dns-name` flag as our host value, otherwise
		// we fallback to using the value specified by the `server` flag as
		// our host value.
		//
		// For a service with only one certificate chain the host value is
		// less important, but for a host with multiple certificate chains
		// having the correct host value is crucial.
		var hostVal string
		switch {

		// We have a resolved IP Address and a sysadmin-specified DNS Name
		// value to use for a SNI-enabled certificate retrieval attempt.
		case expandedHost.Resolved && cfg.DNSName != "":
			hostVal = cfg.DNSName
			certChainSource = fmt.Sprintf(
				"service running on %s (%s) at port %d using host value %q",
				expandedHost.Given,
				ipAddr,
				cfg.Port,
				hostVal,
			)

		// We have a resolved IP Address, but not a sysadmin-specified DNS
		// Name value. We'll use the resolvable name/FQDN for a SNI-enabled
		// certificate retrieval attempt.
		case expandedHost.Resolved && cfg.DNSName == "":
			hostVal = expandedHost.Given
			certChainSource = fmt.Sprintf(
				"service running on %s (%s) at port %d using host value %q",
				expandedHost.Given,
				ipAddr,
				cfg.Port,
				expandedHost.Given,
			)
		default:
			certChainSource = fmt.Sprintf(
				"service running on %s at port %d",
				ipAddr,
				cfg.Port,
			)
		}

		log.Debug().
			Str("server", cfg.Server).
			Str("dns_name", cfg.DNSName).
			Str("ip_address", ipAddr).
			Str("host_value", hostVal).
			Int("port", cfg.Port).
			Msg("Retrieving certificate chain")
		var certFetchErr error
		certChain, certFetchErr = netutils.GetCerts(
			hostVal,
			ipAddr,
			cfg.Port,
			cfg.Timeout(),
			log,
		)
		if certFetchErr != nil {
			log.Error().Err(certFetchErr).Msg(
				"Error fetching certificates chain")

			nagiosExitState.AddError(certFetchErr)
			nagiosExitState.ServiceOutput = fmt.Sprintf(
				"%s: Error fetching certificates from port %d on %s",
				nagios.StateCRITICALLabel,
				cfg.Port,
				cfg.Server,
			)
			nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

			// no need to go any further, we *want* to exit right away; we don't
			// have a connection to the remote server and there isn't anything
			// further we can do
			return

		}

	}

	certsSummary := certs.ChainSummary(
		certChain,
		certsExpireAgeCritical,
		certsExpireAgeWarning,
	)

	// NOTE: Not sure this would ever be reached due to expectations of
	// tls.Dial() that a certificate is present for the connection
	if certsSummary.TotalCertsCount == 0 {
		noCertsErr := fmt.Errorf("no certificates found")
		nagiosExitState.AddError(noCertsErr)
		nagiosExitState.ServiceOutput = fmt.Sprintf(
			"%s: 0 certificates found at port %d on %q",
			nagios.StateCRITICALLabel,
			cfg.Port,
			cfg.Server,
		)
		nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode
		log.Error().Err(noCertsErr).Msg("No certificates found")

		return
	}

	// Prepend a baseline lead-in that summarizes the number of certificates
	// retrieved and from which target host/IP Address.
	defer func() {
		nagiosExitState.LongServiceOutput = fmt.Sprintf(
			"%d certs found for %s%s%s%s",
			certsSummary.TotalCertsCount,
			certChainSource,
			nagios.CheckOutputEOL,
			nagios.CheckOutputEOL,
			nagiosExitState.LongServiceOutput,
		)
	}()

	if certsSummary.TotalCertsCount > 0 {

		hostnameValue := cfg.Server

		// Allow user to explicitly specify which hostname should be used
		// for comparison against the leaf certificate.
		if cfg.DNSName != "" {
			hostnameValue = cfg.DNSName
		}

		// Go 1.17 removed support for the legacy behavior of treating the
		// CommonName field on X.509 certificates as a host name when no
		// Subject Alternative Names are present. Go 1.17 also removed support
		// for re-enabling the behavior by way of adding the value
		// x509ignoreCN=0 to the GODEBUG environment variable.
		//
		// If the SANs list is empty and if requested, we skip hostname
		// verification and log the event.
		switch {
		case len(certChain[0].DNSNames) == 0 &&
			cfg.DisableHostnameVerificationIfEmptySANsList:

			log.Warn().
				Str("hostname", hostnameValue).
				Str("cert_cn", certChain[0].Subject.CommonName).
				Str("sans_entries", fmt.Sprintf("%s", certChain[0].DNSNames)).
				Msg("disabling hostname verification as requested for empty SANs list")

		default:

			// Verify leaf certificate is valid for the provided server FQDN; we
			// make the assumption that the leaf certificate is ALWAYS in position
			// 0 of the chain. Not having the cert in that position is treated as
			// an error condition.
			//
			// Server Name Indication (SNI) support is used to provide the value
			// specified by the `server` flag to the remote server. This is less
			// important for remote hosts with only one certificate, but for a
			// host with multiple certificates it becomes very important to
			// provide the sitename as the value to the `server` flag so that the
			// correct certificate for the connection can be provided.
			verifyErr := certChain[0].VerifyHostname(hostnameValue)

			switch {

			// Go 1.17 removed support for the legacy behavior of treating the
			// CommonName field on X.509 certificates as a host name when no
			// Subject Alternative Names are present. Go 1.17 also removed
			// support for re-enabling the behavior by way of adding the value
			// x509ignoreCN=0 to the GODEBUG environment variable.
			//
			// We attempt to detect this situation in order to supply
			// additional troubleshooting information and guidance to resolve
			// the issue.
			case verifyErr != nil &&
				// TODO: Is there value in matching the specific error string?
				(verifyErr.Error() == certs.X509CertReliesOnCommonName ||
					len(certChain[0].DNSNames) == 0):

				nagiosExitState.AddError(verifyErr)
				nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

				nagiosExitState.ServiceOutput = fmt.Sprintf(
					"%s: Verification of hostname %q failed for first cert in chain",
					nagios.StateCRITICALLabel,
					hostnameValue,
				)
				nagiosExitState.LongServiceOutput =
					"This certificate is missing Subject Alternate Names (SANs)" +
						" and should be replaced." +
						nagios.CheckOutputEOL +
						nagios.CheckOutputEOL +
						"As a temporary workaround, you can" +
						" use v0.5.3 of this plugin, rebuild this plugin" +
						" using Go 1.16 or specify the flag to skip hostname" +
						" verification if the SANs list is found to be empty." +
						nagios.CheckOutputEOL +
						nagios.CheckOutputEOL +
						"See these resources for additional information: " +
						nagios.CheckOutputEOL +
						nagios.CheckOutputEOL +
						" - https://github.com/atc0005/check-cert/issues/276" +
						nagios.CheckOutputEOL +
						" - https://chromestatus.com/feature/4981025180483584" +
						nagios.CheckOutputEOL +
						" - https://bugzilla.mozilla.org/show_bug.cgi?id=1245280"

				return

			// Hostname verification failed for another reason aside from an
			// empty SANs list.
			case verifyErr != nil:
				log.Error().
					Err(verifyErr).
					Str("hostname", hostnameValue).
					Str("cert_cn", certChain[0].Subject.CommonName).
					Str("sans_entries", fmt.Sprintf("%s", certChain[0].DNSNames)).
					Msg("verification of hostname failed for first cert in chain")

				nagiosExitState.AddError(verifyErr)
				nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode

				nagiosExitState.ServiceOutput = fmt.Sprintf(
					"%s: Verification of hostname %q failed for first cert in chain",
					nagios.StateCRITICALLabel,
					hostnameValue,
				)
				nagiosExitState.LongServiceOutput =
					"Consider updating the service check or command " +
						"definition to specify the website FQDN instead of " +
						"the host FQDN as the 'dns-name' (or 'server') flag value. " +
						"E.g., use 'www.example.org' instead of " +
						"'host7.example.com' in order to allow the remote " +
						"server to select the correct certificate instead " +
						"of using the default certificate."

				return

			// Hostname verification succeeded.
			default:

				log.Debug().
					Str("hostname", hostnameValue).
					Str("cert_cn", certChain[0].Subject.CommonName).
					Msg("Verification of hostname succeeded for first cert in chain")

			}
		}

	}

	// check SANS entries if provided via command-line
	if len(cfg.SANsEntries) > 0 {

		// Check for special keyword, skip SANs entry checks if provided
		firstSANsEntry := strings.ToLower(strings.TrimSpace(cfg.SANsEntries[0]))
		if firstSANsEntry != strings.ToLower(strings.TrimSpace(config.SkipSANSCheckKeyword)) {

			mismatched, found, err := certs.CheckSANsEntries(certChain[0], certChain, cfg.SANsEntries)
			if err != nil {

				nagiosExitState.AddError(err)

				nagiosExitState.LongServiceOutput = certs.GenerateCertsReport(
					certsSummary,
					cfg.VerboseOutput,
				)

				nagiosExitState.ServiceOutput = fmt.Sprintf(
					"%s: Mismatch of %d SANs entries for certificate",
					nagios.StateCRITICALLabel,
					mismatched,
				)

				nagiosExitState.ExitStatusCode = nagios.StateCRITICALExitCode
				log.Warn().
					Err(err).
					Int("sans_entries_requested", len(cfg.SANsEntries)).
					Int("sans_entries_found", found).
					Msg("SANs entries mismatch")

				return

			}

			log.Debug().
				Int("sans_entries_requested", len(cfg.SANsEntries)).
				Int("sans_entries_found", found).
				Msg("SANs entries match")
		}
	}

	switch {
	case certsSummary.IsCriticalState() || certsSummary.IsWarningState():

		nagiosExitState.AddError(fmt.Errorf(
			"%d certificates expired or expiring",
			certsSummary.ExpiredCertsCount+certsSummary.ExpiringCertsCount,
		))
		nagiosExitState.LongServiceOutput = certs.GenerateCertsReport(
			certsSummary,
			cfg.VerboseOutput,
		)

		nagiosExitState.ServiceOutput = certs.OneLineCheckSummary(
			certsSummary,
			true,
		)
		nagiosExitState.ExitStatusCode = certsSummary.ServiceState().ExitCode

		log.Error().
			Int("expired_certs", certsSummary.ExpiredCertsCount).
			Int("expiring_certs", certsSummary.ExpiringCertsCount).
			Msg("expired or expiring certs present in chain")

	default:

		nagiosExitState.ServiceOutput = certs.OneLineCheckSummary(
			certsSummary,
			true,
		)

		nagiosExitState.LongServiceOutput = certs.GenerateCertsReport(
			certsSummary,
			cfg.VerboseOutput,
		)
		nagiosExitState.ExitStatusCode = nagios.StateOKExitCode
		log.Debug().Msg("No problems with certificate chain detected")

	}

	// While we provide a flag to skip hostname verification for certs missing
	// SANs list entries, it isn't advisable for long-term use. We note this
	// in the LongServiceOutput as a reminder to sysadmins who may review the
	// output.
	if certsSummary.TotalCertsCount > 0 &&
		len(certChain[0].DNSNames) == 0 &&
		cfg.DisableHostnameVerificationIfEmptySANsList {
		nagiosExitState.LongServiceOutput +=
			nagios.CheckOutputEOL +
				"NOTE: The option to skip hostname verification when" +
				" certificate SANs list is empty has been specified." +
				nagios.CheckOutputEOL +
				nagios.CheckOutputEOL +
				"While viable as a short-term workaround for certificates" +
				" missing SANs list entries, this is not recommended as a" +
				" long-term fix."
	}

}
