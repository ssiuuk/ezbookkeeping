import { ref, computed } from 'vue';
import { defineStore } from 'pinia';

import type {
    ProjectInfoResponse,
    ProjectNewDisplayOrderRequest
} from '@/models/project.ts';

import { handleApiResponse } from '@/lib/api.ts';
import services from '@/lib/services.ts';

export const useProjectsStore = defineStore('projects', () => {
    let loadingPromise: Promise<ProjectInfoResponse[]> | null = null;

    const allProjects = ref<ProjectInfoResponse[]>([]);
    const projectListStateInvalid = ref<boolean>(true);

    const allProjectsMap = computed<Record<string, ProjectInfoResponse>>(() => {
        const result: Record<string, ProjectInfoResponse> = {};

        for (const project of allProjects.value) {
            result[project.id] = project;
        }

        return result;
    });

    const allVisibleProjects = computed<ProjectInfoResponse[]>(() => allProjects.value.filter(project => !project.hidden));

    function updateProjectListInvalidState(invalidState: boolean): void {
        projectListStateInvalid.value = invalidState;
    }

    function resetProjects(): void {
        allProjects.value = [];
        projectListStateInvalid.value = true;
    }

    function loadAllProjects({ force }: { force: boolean }): Promise<ProjectInfoResponse[]> {
        if (!force && !projectListStateInvalid.value) {
            return Promise.resolve(allProjects.value);
        }

        if (loadingPromise) {
            return loadingPromise;
        }

        loadingPromise = handleApiResponse(services.getAllProjects(), 'failed to load project list', 'Unable to retrieve project list').then(projects => {
            allProjects.value = projects;
            projectListStateInvalid.value = false;
            return projects;
        }).finally(() => {
            loadingPromise = null;
        });

        return loadingPromise;
    }

    function getProject({ projectId }: { projectId: string }): Promise<ProjectInfoResponse> {
        return handleApiResponse(services.getProject({ id: projectId }), 'failed to load project info', 'Unable to retrieve project');
    }

    function saveProject({ id, name, comment }: { id?: string, name: string, comment: string }): Promise<ProjectInfoResponse> {
        const promise = id
            ? handleApiResponse(services.modifyProject({ id, name, comment }), 'failed to save project', 'Unable to save project')
            : handleApiResponse(services.addProject({ name, comment }), 'failed to save project', 'Unable to add project');

        return promise.then(project => {
            const index = allProjects.value.findIndex(item => item.id === project.id);

            if (index >= 0) {
                allProjects.value.splice(index, 1, project);
            } else {
                allProjects.value.push(project);
            }

            return project;
        });
    }

    function hideProject({ project, hidden }: { project: ProjectInfoResponse, hidden: boolean }): Promise<boolean> {
        return handleApiResponse(services.hideProject({ id: project.id, hidden }), 'failed to change project visibility', hidden ? 'Unable to hide this project' : 'Unable to unhide this project').then(result => {
            const index = allProjects.value.findIndex(item => item.id === project.id);

            if (index >= 0) {
                allProjects.value.splice(index, 1, { ...allProjects.value[index] as ProjectInfoResponse, hidden: hidden });
            }

            return result;
        });
    }

    function moveProject({ projectId, offset }: { projectId: string, offset: number }): Promise<boolean> {
        const projects = [...allProjects.value];
        const index = projects.findIndex(item => item.id === projectId);
        const newIndex = index + offset;

        if (index < 0 || newIndex < 0 || newIndex >= projects.length) {
            return Promise.resolve(false);
        }

        const [project] = projects.splice(index, 1);
        projects.splice(newIndex, 0, project as ProjectInfoResponse);

        const newDisplayOrders: ProjectNewDisplayOrderRequest[] = projects.map((item, order) => ({
            id: item.id,
            displayOrder: order + 1
        }));

        return handleApiResponse(services.moveProject({ newDisplayOrders }), 'failed to move project', 'Unable to move project').then(result => {
            allProjects.value = projects.map((item, order) => ({ ...item, displayOrder: order + 1 }));
            return result;
        });
    }

    function deleteProject({ project }: { project: ProjectInfoResponse }): Promise<boolean> {
        return handleApiResponse(services.deleteProject({ id: project.id }), 'failed to delete project', 'Unable to delete this project').then(result => {
            allProjects.value = allProjects.value.filter(item => item.id !== project.id);
            return result;
        });
    }

    return {
        // states
        allProjects,
        projectListStateInvalid,
        // computed states
        allProjectsMap,
        allVisibleProjects,
        // functions
        updateProjectListInvalidState,
        resetProjects,
        loadAllProjects,
        getProject,
        saveProject,
        hideProject,
        moveProject,
        deleteProject
    };
});
