package v6_test

import (
	"errors"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/command/commandfakes"
	"code.cloudfoundry.org/cli/command/v6"
	"code.cloudfoundry.org/cli/command/v6/v6fakes"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("deleted-orphaned-routes Command", func() {
	var (
		cmd             v6.DeleteOrphanedRoutesCommand
		testUI          *ui.UI
		fakeConfig      *commandfakes.FakeConfig
		fakeSharedActor *commandfakes.FakeSharedActor
		fakeActor       *v6fakes.FakeDeleteUnmappedRoutesActor
		input           *Buffer
		binaryName      string
		executeErr      error
	)

	BeforeEach(func() {
		input = NewBuffer()
		testUI = ui.NewTestUI(input, NewBuffer(), NewBuffer())
		fakeConfig = new(commandfakes.FakeConfig)
		fakeSharedActor = new(commandfakes.FakeSharedActor)
		fakeActor = new(v6fakes.FakeDeleteUnmappedRoutesActor)

		cmd = v6.DeleteOrphanedRoutesCommand{
			UI:          testUI,
			Config:      fakeConfig,
			SharedActor: fakeSharedActor,
			Actor:       fakeActor,
		}

		binaryName = "faceman"
		fakeConfig.BinaryNameReturns(binaryName)
	})

	JustBeforeEach(func() {
		executeErr = cmd.Execute(nil)
	})

	When("a cloud controller API endpoint is set", func() {
		BeforeEach(func() {
			fakeConfig.TargetReturns("some-url")
		})

		When("checking target fails", func() {
			BeforeEach(func() {
				fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{BinaryName: binaryName})
			})

			It("returns an error", func() {
				Expect(executeErr).To(MatchError(actionerror.NotLoggedInError{BinaryName: "faceman"}))

				Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
				checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
				Expect(checkTargetedOrg).To(BeTrue())
				Expect(checkTargetedSpace).To(BeTrue())
			})
		})

		When("the user is logged in, and org and space are targeted", func() {
			BeforeEach(func() {
				fakeConfig.HasTargetedOrganizationReturns(true)
				fakeConfig.HasTargetedSpaceReturns(true)
				fakeConfig.TargetedSpaceReturns(configv3.Space{
					GUID: "some-space-guid",
					Name: "some-space",
				})
			})

			When("getting the current user returns an error", func() {
				var expectedErr error

				BeforeEach(func() {
					expectedErr = errors.New("EOF")
					fakeConfig.CurrentUserReturns(
						configv3.User{},
						expectedErr)
				})

				It("returns the error", func() {
					Expect(executeErr).To(MatchError(expectedErr))
				})
			})

			When("getting the current user does not return an error", func() {
				BeforeEach(func() {
					fakeConfig.CurrentUserReturns(
						configv3.User{Name: "some-user"},
						nil)
				})

				When("the '-f' flag is provided", func() {
					BeforeEach(func() {
						cmd.Force = true
					})

					It("does not prompt for user confirmation", func() {
						Expect(executeErr).ToNot(HaveOccurred())

						Expect(testUI.Out).ToNot(Say(`Really delete orphaned routes\? \[yN\]:`))
					})
				})

				When("the '-f' flag is not provided", func() {
					When("user is prompted for confirmation", func() {
						BeforeEach(func() {
							_, err := input.Write([]byte("\n"))
							Expect(err).NotTo(HaveOccurred())
						})

						It("displays the interactive prompt", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(testUI.Out).To(Say(`Really delete orphaned routes\? \[yN\]:`))
						})
					})

					When("the user inputs no", func() {
						BeforeEach(func() {
							_, err := input.Write([]byte("n\n"))
							Expect(err).NotTo(HaveOccurred())
						})

						It("does not delete orphaned routes", func() {
							Expect(executeErr).ToNot(HaveOccurred())

							Expect(fakeActor.DeleteUnmappedRoutesCallCount()).To(Equal(0))
						})
					})

					When("the user input is invalid", func() {
						BeforeEach(func() {
							_, err := input.Write([]byte("e\n"))
							Expect(err).NotTo(HaveOccurred())
						})

						It("returns an error", func() {
							Expect(executeErr).To(HaveOccurred())
							Expect(fakeActor.DeleteUnmappedRoutesCallCount()).To(Equal(0))
						})
					})

					When("the user inputs yes", func() {
						BeforeEach(func() {
							_, err := input.Write([]byte("y\n"))
							Expect(err).NotTo(HaveOccurred())
						})

						It("deletes the routes", func() {
							Expect(fakeActor.DeleteUnmappedRoutesCallCount()).To(Equal(1))
							Expect(testUI.Out).To(Say("OK"))
						})

						When("deleting routes returns an error", func() {
							var expectedErr error

							BeforeEach(func() {
								expectedErr = errors.New("EOF")
								fakeActor.DeleteUnmappedRoutesReturns(nil, expectedErr)
							})

							It("returns the error", func() {
								Expect(executeErr).To(MatchError(expectedErr))
							})
						})
					})
				})
			})
		})
	})
})
