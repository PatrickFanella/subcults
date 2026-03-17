/**
 * SceneSettingsPage Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { SceneSettingsPage } from './SceneSettingsPage';
import { useEntityStore } from '../stores/entityStore';
import { useToastStore } from '../stores/toastStore';
import type { Scene } from '../types/scene';

// Mock the stores
vi.mock('../stores/entityStore');
vi.mock('../stores/toastStore');
vi.mock('../stores/authStore', () => ({
  useAuth: () => ({
    user: { did: 'test-user-did', role: 'user' as const },
    logout: vi.fn(),
  }),
}));
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback || key,
  }),
}));

// Mock useParams and useNavigate
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: 'test-scene-id' }),
    useNavigate: () => vi.fn(),
  };
});

describe('SceneSettingsPage', () => {
  const mockScene: Scene = {
    id: 'test-scene-id',
    name: 'Test Scene',
    description: 'Test Description',
    owner_did: 'test-user-did',
    allow_precise: false,
    coarse_geohash: 'u4pruyd',
    tags: ['techno', 'underground'],
    visibility: 'public',
    palette: {
      primary: '#3b82f6',
      secondary: '#8b5cf6',
      accent: '#ec4899',
      background: '#ffffff',
      text: '#000000',
    },
  };

  const mockFetchScene = vi.fn();
  const mockUpdateScene = vi.fn();
  const mockAddToast = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    // Setup store mocks
    (useEntityStore as unknown as ReturnType<typeof vi.fn>).mockImplementation((selector) => {
      const store = {
        fetchScene: mockFetchScene,
        updateScene: mockUpdateScene,
        scene: { scenes: {} },
      };
      return selector(store);
    });

    (useToastStore as unknown as ReturnType<typeof vi.fn>).mockImplementation((selector) => {
      const store = {
        addToast: mockAddToast,
      };
      return selector(store);
    });

    mockFetchScene.mockResolvedValue(mockScene);
  });

  it('renders loading state initially', () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    // Loading state shows skeleton, not text
    const skeletons = screen.getAllByRole('generic');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('loads and displays scene data', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalledWith('test-scene-id');
    });

    await waitFor(() => {
      const nameInput = screen.getByPlaceholderText(/Enter scene name/i);
      expect(nameInput).toHaveValue('Test Scene');
    });
  });

  it('displays scene tags', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('techno')).toBeInTheDocument();
      expect(screen.getByText('underground')).toBeInTheDocument();
    });
  });

  it('displays privacy settings with correct visibility selected', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      const publicRadio = screen.getByLabelText(/Public/i) as HTMLInputElement;
      expect(publicRadio.checked).toBe(true);
    });
  });

  it('displays color customization section', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Visual Customization/i)).toBeInTheDocument();
      expect(screen.getByText(/Primary Color/i)).toBeInTheDocument();
      expect(screen.getByText(/Secondary Color/i)).toBeInTheDocument();
    });
  });

  it('displays member management section', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Members & Alliances/i)).toBeInTheDocument();
      expect(screen.getByText(/View Members/i)).toBeInTheDocument();
    });
  });

  it('displays verification status', async () => {
    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Verification Status/i)).toBeInTheDocument();
      expect(screen.getByText(/Owner Verified/i)).toBeInTheDocument();
    });
  });

  it('shows error toast when scene fetch fails', async () => {
    mockFetchScene.mockRejectedValue(new Error('Failed to load'));

    render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockAddToast).toHaveBeenCalledWith({
        type: 'error',
        message: 'Failed to load scene',
      });
    });
  });

  // Form interaction tests
  it('updates scene name and calls updateScene on save', async () => {
    mockUpdateScene.mockResolvedValue({ ...mockScene, name: 'Updated Name' });

    const { getByPlaceholderText, getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const nameInput = getByPlaceholderText(/Enter scene name/i);
    const saveButton = getByText(/Save Changes/i);

    // Change name
    await waitFor(() => {
      fireEvent.change(nameInput, { target: { value: 'Updated Name' } });
    });

    // Click save
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateScene).toHaveBeenCalledWith(
        'test-scene-id',
        expect.objectContaining({
          name: 'Updated Name',
        })
      );
    });
  });

  it('adds a tag when Enter is pressed', async () => {
    mockUpdateScene.mockResolvedValue(mockScene);

    const { getByPlaceholderText, getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const tagInput = getByPlaceholderText(/e.g., techno/i);

    // Type a new tag
    fireEvent.change(tagInput, { target: { value: 'electronic' } });
    fireEvent.keyDown(tagInput, { key: 'Enter', code: 'Enter' });

    await waitFor(() => {
      expect(screen.getByText('electronic')).toBeInTheDocument();
    });
  });

  it('removes a tag when clicking the remove button', async () => {
    const { getAllByLabelText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText('techno')).toBeInTheDocument();
    });

    const removeButtons = getAllByLabelText(/Remove/i);
    fireEvent.click(removeButtons[0]);

    await waitFor(() => {
      expect(screen.queryByText('techno')).not.toBeInTheDocument();
    });
  });

  it('changes visibility setting', async () => {
    mockUpdateScene.mockResolvedValue({ ...mockScene, visibility: 'private' });

    const { getByLabelText, getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const privateRadio = getByLabelText(/Private/i) as HTMLInputElement;
    const saveButton = getByText(/Save Changes/i);

    fireEvent.click(privateRadio);

    await waitFor(() => {
      expect(privateRadio.checked).toBe(true);
    });

    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockUpdateScene).toHaveBeenCalledWith(
        'test-scene-id',
        expect.objectContaining({
          visibility: 'private',
        })
      );
    });
  });

  it('validates scene name length before saving', async () => {
    const { getByPlaceholderText, getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const nameInput = getByPlaceholderText(/Enter scene name/i);
    const saveButton = getByText(/Save Changes/i);

    // Set name too short
    fireEvent.change(nameInput, { target: { value: 'AB' } });
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockAddToast).toHaveBeenCalledWith({
        type: 'error',
        message: 'Scene name must be between 3 and 64 characters',
      });
    });

    expect(mockUpdateScene).not.toHaveBeenCalled();
  });

  it('validates hex color format before saving', async () => {
    mockUpdateScene.mockResolvedValue(mockScene);

    const { getAllByPlaceholderText, getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const colorInputs = getAllByPlaceholderText(/#[0-9A-Fa-f]{6}/);
    const saveButton = getByText(/Save Changes/i);

    // Set invalid color
    fireEvent.change(colorInputs[0], { target: { value: 'invalid' } });
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockAddToast).toHaveBeenCalledWith({
        type: 'error',
        message: expect.stringContaining('valid hex color'),
      });
    });

    expect(mockUpdateScene).not.toHaveBeenCalled();
  });

  it('disables save button while saving', async () => {
    mockUpdateScene.mockImplementation(() => new Promise((resolve) => setTimeout(resolve, 100)));

    const { getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const saveButton = getByText(/Save Changes/i) as HTMLButtonElement;

    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(saveButton.disabled).toBe(true);
      expect(saveButton.textContent).toContain('Saving');
    });
  });

  it('shows success toast after successful save', async () => {
    mockUpdateScene.mockResolvedValue(mockScene);

    const { getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const saveButton = getByText(/Save Changes/i);

    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockAddToast).toHaveBeenCalledWith({
        type: 'success',
        message: 'Scene settings saved successfully',
      });
    });
  });

  it('shows error toast when save fails', async () => {
    mockUpdateScene.mockRejectedValue(new Error('Save failed'));

    const { getByText } = render(
      <BrowserRouter>
        <SceneSettingsPage />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(mockFetchScene).toHaveBeenCalled();
    });

    const saveButton = getByText(/Save Changes/i);

    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(mockAddToast).toHaveBeenCalledWith({
        type: 'error',
        message: 'Failed to save scene settings',
      });
    });
  });
});
