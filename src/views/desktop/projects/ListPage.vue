<template>
    <main-page-layout no-navbar>
        <template #content>
            <v-row align-content="start">
                <v-col cols="12">
                    <v-card :class="{ 'disabled': loading }">
                        <template #title>
                            <div class="d-flex align-center flex-wrap">
                                <span>{{ tt('Projects') }}</span>
                                <v-btn class="ms-3" density="compact" color="default" variant="outlined"
                                       :disabled="loading || updating" @click="startAdding">
                                    {{ tt('Add Project') }}
                                </v-btn>
                                <v-btn density="compact" color="default" variant="text" size="24"
                                       class="ms-2" :icon="true" :disabled="loading || updating"
                                       :loading="loading" @click="reloadProjects(true)">
                                    <template #loader>
                                        <v-progress-circular indeterminate size="20"/>
                                    </template>
                                    <v-icon :icon="mdiRefresh" size="24" />
                                    <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                                </v-btn>
                                <v-spacer/>
                                <v-switch class="flex-grow-0" density="compact" color="primary" hide-details
                                          :disabled="loading || updating || !hasHiddenProjects"
                                          :label="tt('Show Hidden Projects')"
                                          v-model="showHidden"/>
                            </div>
                        </template>

                        <v-card-text class="pt-0">
                            <span class="text-body-medium text-medium-emphasis">{{ tt('Group expenses and income from different categories into a project, for example a house renovation, and see what it cost in total.') }}</span>
                        </v-card-text>

                        <v-table class="text-no-wrap">
                            <thead>
                            <tr>
                                <th>{{ tt('Project Name') }}</th>
                                <th>{{ tt('Description') }}</th>
                                <th class="text-right">{{ tt('Transactions') }}</th>
                                <th>{{ tt('Date Range') }}</th>
                                <th class="text-right">{{ tt('Operation') }}</th>
                            </tr>
                            </thead>
                            <tbody>
                            <tr v-if="!loading && !projects.length">
                                <td colspan="5">{{ tt('No available project') }}</td>
                            </tr>
                            <tr :key="project.id" :class="{ 'text-disabled': project.hidden }" v-for="project in projects">
                                <td>
                                    <router-link class="project-name-link" :to="`/projects/report?id=${project.id}`">
                                        <v-icon class="me-2" :icon="mdiBriefcaseOutline" size="20" />
                                        <span>{{ project.name }}</span>
                                    </router-link>
                                    <v-chip class="ms-2" size="x-small" v-if="project.hidden">{{ tt('Hidden') }}</v-chip>
                                </td>
                                <td class="project-comment-column text-truncate">{{ project.comment }}</td>
                                <td class="text-right">{{ project.transactionCount }}</td>
                                <td>{{ getProjectDateRangeText(project) }}</td>
                                <td class="text-right">
                                    <v-btn density="comfortable" variant="text" :icon="true"
                                           :to="`/projects/report?id=${project.id}`">
                                        <v-icon :icon="mdiChartBoxOutline" />
                                        <v-tooltip activator="parent">{{ tt('Project Report') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn density="comfortable" variant="text" :icon="true"
                                           :disabled="updating" @click="startEditing(project)">
                                        <v-icon :icon="mdiPencilOutline" />
                                        <v-tooltip activator="parent">{{ tt('Edit') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn density="comfortable" variant="text" :icon="true"
                                           :disabled="updating" @click="hide(project, !project.hidden)">
                                        <v-icon :icon="project.hidden ? mdiEyeOutline : mdiEyeOffOutline" />
                                        <v-tooltip activator="parent">{{ project.hidden ? tt('Show') : tt('Hide') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn density="comfortable" variant="text" :icon="true"
                                           :disabled="updating || isFirstProject(project)" @click="move(project, -1)">
                                        <v-icon :icon="mdiArrowUp" />
                                        <v-tooltip activator="parent">{{ tt('Move Up') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn density="comfortable" variant="text" :icon="true"
                                           :disabled="updating || isLastProject(project)" @click="move(project, 1)">
                                        <v-icon :icon="mdiArrowDown" />
                                        <v-tooltip activator="parent">{{ tt('Move Down') }}</v-tooltip>
                                    </v-btn>
                                    <v-btn density="comfortable" variant="text" color="error" :icon="true"
                                           :disabled="updating" @click="remove(project)">
                                        <v-icon :icon="mdiDeleteOutline" />
                                        <v-tooltip activator="parent">{{ tt('Delete') }}</v-tooltip>
                                    </v-btn>
                                </td>
                            </tr>
                            </tbody>
                        </v-table>
                    </v-card>
                </v-col>
            </v-row>
        </template>
    </main-page-layout>

    <v-dialog width="560" :persistent="updating" :model-value="!!editingProject" @update:model-value="onEditDialogChanged">
        <v-card class="pa-2 pa-sm-4 pa-md-4" v-if="editingProject">
            <template #title>
                <h4 class="text-h4">{{ editingProject.id ? tt('Edit Project') : tt('Add Project') }}</h4>
            </template>
            <v-card-text>
                <v-row>
                    <v-col cols="12">
                        <v-text-field
                            type="text"
                            autocomplete="off"
                            persistent-placeholder
                            :disabled="updating"
                            :label="tt('Project Name')"
                            :placeholder="tt('Project Name')"
                            v-model="editingProject.name"
                            @keyup.enter="save"
                        />
                    </v-col>
                    <v-col cols="12">
                        <v-textarea
                            type="text"
                            autocomplete="off"
                            persistent-placeholder
                            rows="3"
                            :disabled="updating"
                            :label="tt('Description')"
                            :placeholder="tt('Your project description (optional)')"
                            v-model="editingProject.comment"
                        />
                    </v-col>
                </v-row>
            </v-card-text>
            <v-card-text>
                <div class="w-100 d-flex justify-center flex-wrap mt-sm-1 mt-md-2 gap-4">
                    <v-btn :disabled="updating || !editingProject.name.trim()" @click="save">
                        {{ tt('Save') }}
                        <v-progress-circular indeterminate size="22" class="ms-2" v-if="updating"></v-progress-circular>
                    </v-btn>
                    <v-btn color="secondary" variant="tonal" :disabled="updating" @click="cancelEditing">{{ tt('Cancel') }}</v-btn>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useProjectListPageBase } from '@/views/base/projects/ProjectPageBase.ts';

import type { ProjectInfoResponse } from '@/models/project.ts';

import {
    mdiRefresh,
    mdiBriefcaseOutline,
    mdiChartBoxOutline,
    mdiPencilOutline,
    mdiEyeOutline,
    mdiEyeOffOutline,
    mdiArrowUp,
    mdiArrowDown,
    mdiDeleteOutline
} from '@mdi/js';

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;

const { tt } = useI18n();
const {
    loading,
    updating,
    showHidden,
    editingProject,
    projects,
    hasHiddenProjects,
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
} = useProjectListPageBase();

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');

function showError(error: unknown): void {
    if (!(error as { processed?: boolean }).processed) {
        snackbar.value?.showError(error as { message: string });
    }
}

function reloadProjects(force: boolean): void {
    reload(force).catch(showError);
}

function onEditDialogChanged(value: boolean): void {
    if (!value) {
        cancelEditing();
    }
}

function save(): void {
    const isNew = !editingProject.value?.id;

    saveEditingProject().then(project => {
        if (project) {
            snackbar.value?.showMessage(isNew ? 'You have added a new project' : 'You have saved this project');
        }
    }).catch(showError);
}

function hide(project: ProjectInfoResponse, hidden: boolean): void {
    hideProject(project, hidden).catch(showError);
}

function move(project: ProjectInfoResponse, offset: number): void {
    moveProject(project, offset).catch(showError);
}

function remove(project: ProjectInfoResponse): void {
    confirmDialog.value?.open('Are you sure you want to delete this project? Its transactions will be kept but no longer belong to it.', { color: 'error' }).then(() => {
        deleteProject(project).then(() => {
            snackbar.value?.showMessage('You have deleted this project');
        }).catch(showError);
    });
}

reloadProjects(true);
</script>

<style>
.project-name-link {
    display: inline-flex;
    align-items: center;
    color: inherit;
    text-decoration: none;
}

.project-name-link:hover {
    text-decoration: underline;
}

.project-comment-column {
    max-width: 320px;
}
</style>
