import { type SubmitHandler, useForm } from "@formisch/react";
import { useNavigate } from "@tanstack/react-router";
import * as v from "valibot";
import { useLoginWithEmailApi } from "#/api/auth";

export const LoginSchema = v.object({
	email: v.pipe(
		v.string(),
		v.email(),
		v.maxLength(255, "Email must be a maximum of 255 characters."),
	),
	password: v.pipe(v.string(), v.minLength(1, "Password is required.")),
});

export const useLoginForm = () => {
	const navigate = useNavigate();
	const loginMutation = useLoginWithEmailApi();
	const loginForm = useForm({
		schema: LoginSchema,
		initialInput: {
			email: "",
			password: "",
		},
	});

	const handleSubmit: SubmitHandler<typeof LoginSchema> = (output) => {
		void (async () => {
			await loginMutation.mutateAsync(output, {onSuccess: (data) => {
				console.log(data)
			}});
			await navigate({ to: "/" });
		})().catch(() => undefined);
	};

	return {
		loginForm,
		handleSubmit,
		error: loginMutation.error,
		isPending: loginMutation.isPending,
	};
};
