import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { useProjectsStore } from '@/stores/project.ts';

import type { ProjectInfoResponse } from '@/models/project.ts';

import { parseDateTimeFromUnixTime } from '@/lib/datetime.ts';

export interface ProjectEditingModel {
    id?: string;
    name: string;
    comment: string;
}

export function useProjectDateRangeText() {
    const { tt, formatDateTimeToShortDate } = useI18n();

    function getProjectDateRangeText(project?: ProjectInfoResponse | null): string {
        if (!project || !project.transactionCount || !project.firstTransactionTime || !project.lastTransactionTime) {
            return tt('No transactions');
        }

        const firstDate = formatDateTimeToShortDate(parseDateTimeFromUnixTime(project.firstTransactionTime));
        const lastDate = formatDateTimeToShortDate(parseDateTimeFromUnixTime(project.lastTransactionTime));

        if (firstDate === lastDate) {
            return firstDate;
        }

        return `${firstDate} - ${lastDate}`;
    }

    return {
        getProjectDateRangeText
    };
}

export function useProjectListPageBase() {
    const { getProjectDateRangeText } = useProjectDateRangeText();

    const projectsStore = useProjectsStore();

    const loading = ref<boolean>(true);
    const updating = ref<boolean>(false);
    const showHidden = ref<boolean>(false);
    const editingProject = ref<ProjectEditingModel | null>(null);

    const projects = computed<ProjectInfoResponse[]>(() => projectsStore.allProjects.filter(project => showHidden.value || !project.hidden));
    const hasHiddenProjects = computed<boolean>(() => projectsStore.allProjects.some(project => project.hidden));

    function reload(force: boolean): Promise<ProjectInfoResponse[]> {
        loading.value = true;

        return projectsStore.loadAllProjects({ force }).finally(() => {
            loading.value = false;
        });
    }

    function startAdding(): void {
        editingProject.value = { name: '', comment: '' };
    }

    function startEditing(project: ProjectInfoResponse): void {
        editingProject.value = { id: project.id, name: project.name, comment: project.comment };
    }

    function cancelEditing(): void {
        editingProject.value = null;
    }

    function withUpdating<T>(promise: () => Promise<T>): Promise<T> {
        updating.value = true;

        return promise().finally(() => {
            updating.value = false;
        });
    }

    function saveEditingProject(): Promise<ProjectInfoResponse | null> {
        const project = editingProject.value;

        if (!project || !project.name.trim()) {
            return Promise.resolve(null);
        }

        return withUpdating(() => projectsStore.saveProject({ id: project.id, name: project.name.trim(), comment: project.comment.trim() })).then(result => {
            editingProject.value = null;
            return result;
        });
    }

    function hideProject(project: ProjectInfoResponse, hidden: boolean): Promise<boolean> {
        return withUpdating(() => projectsStore.hideProject({ project, hidden }));
    }

    function moveProject(project: ProjectInfoResponse, offset: number): Promise<boolean> {
        return withUpdating(() => projectsStore.moveProject({ projectId: project.id, offset }));
    }

    function deleteProject(project: ProjectInfoResponse): Promise<boolean> {
        return withUpdating(() => projectsStore.deleteProject({ project }));
    }

    function isFirstProject(project: ProjectInfoResponse): boolean {
        return projectsStore.allProjects[0]?.id === project.id;
    }

    function isLastProject(project: ProjectInfoResponse): boolean {
        return projectsStore.allProjects[projectsStore.allProjects.length - 1]?.id === project.id;
    }

    return {
        // states
        loading,
        updating,
        showHidden,
        editingProject,
        // computed states
        projects,
        hasHiddenProjects,
        // functions
        reload,
        startAdding,
        startEditing,
        cancelEditing,
        saveEditingProject,
        hideProject,
        moveProject,
        deleteProject,
        isFirstProject,
        isLastProject,
        getProjectDateRangeText
    };
}
