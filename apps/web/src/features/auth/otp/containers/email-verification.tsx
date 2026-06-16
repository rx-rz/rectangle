import { useNavigate, useSearch } from "@tanstack/react-router";
import { REGEXP_ONLY_DIGITS } from "input-otp";
import { useEffect, useState } from "react";
import { useSendOTPApi, useVerifyOTPApi } from "#/api/auth";
import { Button } from "#/components/ui/button";
import {
	InputOTP,
	InputOTPGroup,
	InputOTPSlot,
} from "@/components/ui/input-otp";

const RESEND_COOLDOWN = 60;
const OTP_SLOT_KEYS = ["first", "second", "third", "fourth", "fifth", "sixth"];

export const EmailVerification = () => {
	const navigate = useNavigate();
	const { email } = useSearch({ from: "/auth/verify-email" });
	const [otpCode, setOtpCode] = useState("");
	const [resendCooldown, setResendCooldown] = useState(RESEND_COOLDOWN);
	const verifyOTPMutation = useVerifyOTPApi();
	const getOTPMutation = useSendOTPApi();

	const canResend = resendCooldown === 0;
	const error = verifyOTPMutation.error ?? getOTPMutation.error;
	const isPending = verifyOTPMutation.isPending || getOTPMutation.isPending;

	useEffect(() => {
		if (resendCooldown === 0) return;

		const timeoutId = setTimeout(() => {
			setResendCooldown((prev) => prev - 1);
		}, 1000);

		return () => clearTimeout(timeoutId);
	}, [resendCooldown]);

	const handleResendOTP = async () => {
		if (!canResend || !email) return;

		await getOTPMutation.mutateAsync({ email });

		setResendCooldown(RESEND_COOLDOWN);
	};

	const handleVerifyOTP = async () => {
		if (!email || otpCode.length !== 6) return;

		try {
			await verifyOTPMutation.mutateAsync({ email, code: otpCode });
			await navigate({ to: "/" });
		} catch {
			return;
		}
	};

	return (
		<div className="mt-12">
			<p className="text-xs text-primary uppercase">
				{"// Enter verification code"}
			</p>

			<p className="opacity-70 my-4">
				Enter the verification code sent to <br />
				{email || "your email address"}
			</p>

			<InputOTP
				maxLength={6}
				value={otpCode}
				onChange={setOtpCode}
				pattern={REGEXP_ONLY_DIGITS}
			>
				<InputOTPGroup className="my-8 flex gap-2 w-full justify-between">
					{OTP_SLOT_KEYS.map((key, index) => (
						<InputOTPSlot key={key} index={index} className="size-18 text-xl" />
					))}
				</InputOTPGroup>
			</InputOTP>

			{error && (
				<p className="mb-4 text-destructive text-sm">{error.message}</p>
			)}

			<Button
				className="w-full uppercase"
				disabled={!email || otpCode.length !== 6 || isPending}
				onClick={handleVerifyOTP}
			>
				{verifyOTPMutation.isPending ? "Verifying..." : "Proceed"}
			</Button>

			<button
				type="button"
				onClick={handleResendOTP}
				disabled={!canResend}
				className="mt-6 w-full text-xs uppercase opacity-70 disabled:cursor-not-allowed disabled:opacity-40"
			>
				{canResend ? "Resend code" : `Resend code in ${resendCooldown}s`}
			</button>
		</div>
	);
};
