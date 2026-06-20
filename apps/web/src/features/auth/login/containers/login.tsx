import { Form, Field as FormischField } from "@formisch/react";
import { Button } from "#/components/ui/button";
import { Field, FieldError, FieldLabel } from "#/components/ui/field";
import { Input } from "#/components/ui/input";
import { useLoginForm } from "../form/login-form";

export const LoginForm = () => {
	const { loginForm, handleSubmit, error, isPending } = useLoginForm();

	return (
		<div className="h-full mt-8">
			<Form
				of={loginForm}
				id="rectangle-login"
				onSubmit={handleSubmit}
				className=" flex flex-col gap-6"
			>
				<FormischField of={loginForm} path={["email"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="email">Work Email</FieldLabel>
							<Input
								{...field.props}
								id="email"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="email"
								type="email"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				<FormischField of={loginForm} path={["password"]}>
					{(field) => (
						<Field data-invalid={field.errors !== null}>
							<FieldLabel htmlFor="password">Password</FieldLabel>
							<Input
								{...field.props}
								id="password"
								value={field.input ?? ""}
								aria-invalid={field.errors !== null}
								autoComplete="current-password"
								type="password"
							/>
							{field.errors && (
								<FieldError
									errors={field.errors.map((message) => ({ message }))}
								/>
							)}
						</Field>
					)}
				</FormischField>
				{error && <p className="text-destructive text-sm">{error.message}</p>}
				<Button
					type="submit"
					form="rectangle-login"
					className="mt-4"
					disabled={isPending}
				>
					{isPending ? "Signing In..." : "Sign In"}
				</Button>
			</Form>
		</div>
	);
};
