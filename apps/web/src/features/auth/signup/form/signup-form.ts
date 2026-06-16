import { type SubmitHandler, useForm } from "@formisch/react";
import { useNavigate } from "@tanstack/react-router";
import * as v from "valibot";
import { useSendOTPApi, useSignupWithEmailApi } from "#/api/auth";
export const SignupSchema = v.pipe(
	v.object({
		name: v.pipe(
			v.string(),
			v.minLength(5, "Name must be at least 5 characters"),
			v.maxLength(120, "Name must be a maximum of 120 characters."),
		),
		email: v.pipe(
			v.string(),
			v.email(),
			v.maxLength(255, "Email must be a maximum of 255 characters."),
		),
		password: v.pipe(
			v.string(),
			v.minLength(8, "Password must be at least 8 characters."),
		),
		confirmPassword: v.pipe(
			v.string(),
			v.minLength(8, "Confirm password must be at least 8 characters."),
		),
	}),
	v.forward(
		v.check(
			({ password, confirmPassword }) => password === confirmPassword,
			"Passwords must match.",
		),
		["confirmPassword"],
	),
);

export const useSignupForm = () => {
	const navigate = useNavigate();
	const signupMutation = useSignupWithEmailApi();
	const getOTPMutation = useSendOTPApi();
	const signupForm = useForm({
		schema: SignupSchema,
		initialInput: {
			confirmPassword: "",
			email: "",
			name: "",
			password: "",
		},
	});

	const handleSubmit: SubmitHandler<typeof SignupSchema> = (output) => {
		void (async () => {
			await signupMutation.mutateAsync({
				email: output.email,
				name: output.name,
				password: output.password,
			});
			await getOTPMutation.mutateAsync({ email: output.email });
			await navigate({
				to: "/auth/verify-email",
				search: { email: output.email },
			});
		})().catch(() => undefined);
	};

	return {
		signupForm,
		handleSubmit,
		error: signupMutation.error ?? getOTPMutation.error,
		isPending: signupMutation.isPending || getOTPMutation.isPending,
	};
};
