// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/gardener/auditlog-forwarder/pkg/apis/config/v1alpha1"
)

var _ = Describe("Defaults", func() {
	var (
		obj *AuditlogForwarder
	)

	BeforeEach(func() {
		obj = &AuditlogForwarder{}
	})

	Describe("#SetDefaults_AuditlogForwarder", func() {
		It("should default the log level and format", func() {
			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Log.Level).To(Equal(LogLevelInfo))
			Expect(obj.Log.Format).To(Equal(LogFormatJSON))
		})

		It("should default the server port", func() {
			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Server.Port).To(Equal(uint(10443)))
		})

		It("should not override existing values", func() {
			obj.Log.Level = LogLevelDebug
			obj.Log.Format = LogFormatText
			obj.Server.Port = 8080

			SetDefaults_AuditlogForwarder(obj)

			Expect(obj.Log.Level).To(Equal(LogLevelDebug))
			Expect(obj.Log.Format).To(Equal(LogFormatText))
			Expect(obj.Server.Port).To(Equal(uint(8080)))
		})
	})

	Describe("#SetDefaults_Log", func() {
		var (
			logConfig *Log
		)

		BeforeEach(func() {
			logConfig = &Log{}
		})

		It("should default the level to info", func() {
			SetDefaults_Log(logConfig)

			Expect(logConfig.Level).To(Equal(LogLevelInfo))
		})

		It("should default the format to json", func() {
			SetDefaults_Log(logConfig)

			Expect(logConfig.Format).To(Equal(LogFormatJSON))
		})

		It("should not override existing values", func() {
			logConfig.Level = LogLevelDebug
			logConfig.Format = LogFormatText

			SetDefaults_Log(logConfig)

			Expect(logConfig.Level).To(Equal(LogLevelDebug))
			Expect(logConfig.Format).To(Equal(LogFormatText))
		})
	})

	Describe("#SetDefaults_Server", func() {
		var (
			serverConfig *Server
		)

		BeforeEach(func() {
			serverConfig = &Server{}
		})

		It("should default the port to 10443", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Port).To(Equal(uint(10443)))
		})

		It("should not override existing port value", func() {
			serverConfig.Port = 8080

			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Port).To(Equal(uint(8080)))
		})

		It("should not set defaults for address (should remain empty)", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.Address).To(BeEmpty())
		})

		It("should not set defaults for TLS configuration", func() {
			SetDefaults_Server(serverConfig)

			Expect(serverConfig.TLS.CertFile).To(BeEmpty())
			Expect(serverConfig.TLS.KeyFile).To(BeEmpty())
		})
	})
})
