import { useQuery } from "@tanstack/react-query";

import { API_ROUTES } from "../api-routes";
import { api } from "../index";
import { queries } from "../queries";
import type { AuthUser } from "./signup-with-email";

export type GithubConnection = {
	connected: boolean;
	can_import_projects: boolean;
};

export type MeResponse = {
	user: AuthUser;
	connections: {
		github: GithubConnection;
	};
};

export const getMeApi = async () => {
	return await api.request<MeResponse>(API_ROUTES.me);
};

export const useGetMeApi = () => {
	const query = useQuery({
		queryKey: queries.auth.me.queryKey,
		queryFn: getMeApi,
		retry: false,
	});

	const { data, error, isFetching, isSuccess } = query;

	return {
		...query,
		data,
		error,
		isFetching,
		isSuccess,
	};
};
